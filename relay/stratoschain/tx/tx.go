package tx

import (
	ed25519crypto "crypto/ed25519"
	"encoding/hex"
	"sync"

	"github.com/pkg/errors"

	"github.com/cosmos/cosmos-sdk/client"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/stratosnet/sds/relay/stratoschain/grpc"
	relaytypes "github.com/stratosnet/sds/relay/types"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	utilsecp256k1 "github.com/stratosnet/sds/utils/crypto/secp256k1"
)

func BuildTxBytes(protoConfig client.TxConfig, txBuilder client.TxBuilder, chainId string, unsignedMsgs []*relaytypes.UnsignedMsg) ([]byte, error) {
	filteredMsgs := filterInvalidSignatures(unsignedMsgs)          // Filter msgs with invalid signatures
	accountInfos := fetchAllAccountInfos(filteredMsgs)             // Fetch account info from stratos-chain for each signature
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
			if len(signature.Address) == 0 || len(signature.PrivateKey) == 0 {
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
		switch signatureKey.Type {
		case relaytypes.SignatureEd25519:
			if len(signatureKey.PrivateKey) != ed25519crypto.PrivateKeySize {
				return []byte{}, errors.New("ed25519 private key has wrong length: " + hex.EncodeToString(signatureKey.PrivateKey))
			}
			pubkey = ed25519.PrivKeyBytesToSdkPubKey(signatureKey.PrivateKey)
		default:
			pubkey = utilsecp256k1.PrivKeyToSdkPrivKey(signatureKey.PrivateKey).PubKey()
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
			return []byte{}, err
		}
	}

	// Second round: all signer infos are set, so each signer can sign.
	for _, signatureKey := range signaturesToDo {
		signerData := authsigning.SignerData{
			ChainID:       chainId,
			AccountNumber: signatureKey.AccountNum,
			Sequence:      signatureKey.AccountSequence,
		}

		var privKey cryptotypes.PrivKey
		switch signatureKey.Type {
		case relaytypes.SignatureEd25519:
			privKey = ed25519.PrivKeyBytesToSdkPrivKey(signatureKey.PrivateKey)
		default:
			privKey = utilsecp256k1.PrivKeyToSdkPrivKey(signatureKey.PrivateKey)
		}
		sigV2, err := clienttx.SignWithPrivKey(
			protoConfig.SignModeHandler().DefaultMode(), signerData,
			txBuilder, privKey, protoConfig, signerData.Sequence)
		if err != nil {
			return []byte{}, err
		}
		err = txBuilder.SetSignatures(sigV2)
		if err != nil {
			return []byte{}, err
		}
	}
	txBytes, err := protoConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return []byte{}, err
	}

	return txBytes, nil
}
