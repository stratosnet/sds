package stratoschain

import (
	ed25519crypto "crypto/ed25519"
	"encoding/hex"
	"sync"

	"github.com/cosmos/cosmos-sdk/client"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/hd"

	"github.com/cosmos/cosmos-sdk/crypto/ledger"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/relay/stratoschain/grpc"
	relaytypes "github.com/stratosnet/sds/relay/types"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	utilsecp256k1 "github.com/stratosnet/sds/utils/crypto/secp256k1"
	stchaintypes "github.com/stratosnet/stratos-chain/types"
)

func BuildTxBytes(protoConfig client.TxConfig, txBuilder client.TxBuilder, chainId string, unsignedMsgs []*relaytypes.UnsignedMsg) ([]byte, error) {
	filteredMsgs := filterInvalidSignatures(unsignedMsgs)          // Filter out msgs with invalid signature key (only check the address)
	accountInfos := fetchAllAccountInfos(filteredMsgs)             // Fetch account info from stratos-chain for verifying the signature key
	updatedMsgs := updateSignatureKeys(filteredMsgs, accountInfos) // Update signatureKeys for each msg

	if len(updatedMsgs) != len(unsignedMsgs) {
		utils.ErrorLogf("BuildTxBytes couldn't build all the msgs provided (success: %v  invalid_signature: %v  missing_account_infos: %v",
			len(updatedMsgs), len(unsignedMsgs)-len(filteredMsgs), len(filteredMsgs)-len(updatedMsgs))
	}

	return buildAndSignStdTx(protoConfig, txBuilder, chainId, updatedMsgs)
}

func filterInvalidSignatures(msgs []*relaytypes.UnsignedMsg) []*relaytypes.UnsignedMsg {
	var filteredMsgs []*relaytypes.UnsignedMsg
	for _, msg := range msgs {
		invalidSignature := false
		for _, signature := range msg.SignatureKeys {
			// when private key is not presented, the Tx will be signed by hardware device
			if len(signature.Address) == 0 {
				invalidSignature = true
				break
			}
		}
		if invalidSignature {
			continue
		}
		filteredMsgs = append(filteredMsgs, msg)
	}
	return filteredMsgs
}

func fetchAllAccountInfos(msgs []*relaytypes.UnsignedMsg) map[string]*authtypes.BaseAccount {
	// Gather all accounts to fetch
	accountsToFetch := make(map[string]bool)
	for _, msg := range msgs {
		for _, signatureKey := range msg.SignatureKeys {
			accountsToFetch[signatureKey.Address] = true
		}
	}

	// Fetch all accounts in parallel
	results := make(map[string]*authtypes.BaseAccount)
	mutex := sync.Mutex{}
	wg := sync.WaitGroup{}
	for account := range accountsToFetch {
		wg.Add(1)
		go func(walletAddress string) {
			defer wg.Done()

			baseAccount, err := grpc.QueryAccount(walletAddress)
			if err == nil {
				mutex.Lock()
				results[walletAddress] = baseAccount
				mutex.Unlock()
			} else {
				utils.ErrorLogf("Error when fetching account info for wallet %v: %v", walletAddress, err.Error())
			}
		}(account)
	}
	wg.Wait()
	return results
}

func updateSignatureKeys(msgs []*relaytypes.UnsignedMsg, accountInfos map[string]*authtypes.BaseAccount) []*relaytypes.UnsignedMsg {
	var filteredMsgs []*relaytypes.UnsignedMsg
	for _, msg := range msgs {
		missingInfos := false
		for i, signatureKey := range msg.SignatureKeys {
			info, found := accountInfos[signatureKey.Address]
			if info == nil || !found {
				missingInfos = true
				break
			}
			signatureKey.AccountNum = info.GetAccountNumber()
			signatureKey.AccountSequence = info.GetSequence()
			msg.SignatureKeys[i] = signatureKey
		}
		if missingInfos {
			continue
		}

		filteredMsgs = append(filteredMsgs, msg)
	}

	return filteredMsgs
}

// signWithLedgerPrivKey this is a Ledger version of clienttx.SignWithPrivKey. Instead of using cryptotypes.PrivKey as
// the private key for sign, a LedgerPrivKey is used.
func signWithLedgerPrivKey(
	signMode signingtypes.SignMode, signerData authsigning.SignerData,
	txBuilder client.TxBuilder, priv cryptotypes.LedgerPrivKey, txConfig client.TxConfig,
	accSeq uint64,
) (signingtypes.SignatureV2, error) {
	var sigV2 signingtypes.SignatureV2

	// Generate the bytes to be signed.
	signBytes, err := txConfig.SignModeHandler().GetSignBytes(signMode, signerData, txBuilder.GetTx())
	if err != nil {
		err = errors.Wrap(err, "failed getting sign baytes, ")
		return sigV2, err
	}

	// Sign those bytes
	signature, err := priv.Sign(signBytes)
	if err != nil {
		err = errors.Wrap(err, "failed signing signBytes, ")
		return sigV2, err
	}

	// Construct the SignatureV2 struct
	sigData := signingtypes.SingleSignatureData{
		SignMode:  signMode,
		Signature: signature,
	}

	pubkey, err := utilsecp256k1.PubKeyToSdkPubKey(priv.PubKey().Bytes())
	if err != nil {
		return sigV2, err
	}

	sigV2 = signingtypes.SignatureV2{
		PubKey:   pubkey,
		Data:     &sigData,
		Sequence: accSeq,
	}

	return sigV2, nil
}

func buildAndSignStdTx(protoConfig client.TxConfig, txBuilder client.TxBuilder, chainId string, unsignedMsgs []*relaytypes.UnsignedMsg) ([]byte, error) {
	if len(unsignedMsgs) == 0 {
		return nil, errors.New("cannot build tx: no msgs to sign")
	}
	// Collect list of signatures to do. Must match order of GetSigners() method in cosmos-sdk's stdtx.go
	var signaturesToDo []relaytypes.SignatureKey
	signersSeen := make(map[string]bool)
	for _, msg := range unsignedMsgs {
		for _, signaturekey := range msg.SignatureKeys {
			if !signersSeen[signaturekey.Address] {
				signersSeen[signaturekey.Address] = true
				signaturesToDo = append(signaturesToDo, signaturekey)
			}
		}
	}

	var sigsV2 []signingtypes.SignatureV2
	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	for _, signatureKey := range signaturesToDo {
		var pubkey cryptotypes.PubKey
		if signatureKey.PrivateKey == nil {
			path := *hd.NewFundraiserParams(0, stchaintypes.CoinType, 0)
			privkey, err := ledger.NewPrivKeySecp256k1Unsafe(path)
			if err != nil {
				return nil, err
			}
			pubkey, err = utilsecp256k1.PubKeyToSdkPubKey(privkey.PubKey().Bytes())
			if err != nil {
				return nil, err
			}
		} else {
			switch signatureKey.Type {
			case relaytypes.SignatureEd25519:
				if len(signatureKey.PrivateKey) != ed25519crypto.PrivateKeySize {
					return []byte{}, errors.New("ed25519 private key has wrong length: " + hex.EncodeToString(signatureKey.PrivateKey))
				}
				pubkey = ed25519.PrivKeyBytesToSdkPubKey(signatureKey.PrivateKey)
			default:
				pubkey = utilsecp256k1.PrivKeyToSdkPrivKey(signatureKey.PrivateKey).PubKey()
			}
		}

		sigV2 := signingtypes.SignatureV2{
			PubKey: pubkey,
			Data: &signingtypes.SingleSignatureData{
				SignMode:  protoConfig.SignModeHandler().DefaultMode(),
				Signature: nil,
			},
			Sequence: signatureKey.AccountSequence,
		}

		sigsV2 = append(sigsV2, sigV2)
		err := txBuilder.SetSignatures(sigsV2...)
		if err != nil {
			return []byte{}, errors.Wrap(err, "failed setting signatures, ")
		}
	}

	// Second round: all signer infos are set, so each signer can sign.
	for _, signatureKey := range signaturesToDo {
		signerData := authsigning.SignerData{
			ChainID:       chainId,
			AccountNumber: signatureKey.AccountNum,
			Sequence:      signatureKey.AccountSequence,
		}
		var sigV2 signingtypes.SignatureV2
		var err error

		if signatureKey.PrivateKey == nil {
			var priv cryptotypes.LedgerPrivKey
			path := *hd.NewFundraiserParams(0, stchaintypes.CoinType, 0)
			priv, err = ledger.NewPrivKeySecp256k1Unsafe(path)
			if err != nil {
				return nil, errors.Wrap(err, "failed opening the key, ")
			}
			sigV2, err = signWithLedgerPrivKey(protoConfig.SignModeHandler().DefaultMode(), signerData, txBuilder, priv, protoConfig, signerData.Sequence)
			if err != nil {
				return []byte{}, errors.Wrap(err, "failed signing, ")
			}
		} else {
			var privKey cryptotypes.PrivKey
			switch signatureKey.Type {
			case relaytypes.SignatureEd25519:
				privKey = ed25519.PrivKeyBytesToSdkPrivKey(signatureKey.PrivateKey)
			default:
				privKey = utilsecp256k1.PrivKeyToSdkPrivKey(signatureKey.PrivateKey)
			}
			sigV2, err = clienttx.SignWithPrivKey(
				protoConfig.SignModeHandler().DefaultMode(), signerData,
				txBuilder, privKey, protoConfig, signerData.Sequence)
			if err != nil {
				return []byte{}, errors.Wrap(err, "failed signing, ")
			}
		}
		err = txBuilder.SetSignatures(sigV2)
		if err != nil {
			return []byte{}, errors.Wrap(err, "failed setting back the signatures, ")
		}
	}
	txBytes, err := protoConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return []byte{}, errors.Wrap(err, "failed encoding tx, ")
	}

	return txBytes, nil
}
