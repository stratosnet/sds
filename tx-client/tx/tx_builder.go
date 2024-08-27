package tx

import (
	ed25519crypto "crypto/ed25519"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	authv1beta1 "cosmossdk.io/api/cosmos/auth/v1beta1"
	basev1beta1 "cosmossdk.io/api/cosmos/base/v1beta1"
	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"

	"github.com/stratosnet/sds/framework/crypto/ed25519"
	"github.com/stratosnet/sds/framework/crypto/secp256k1"
	fwcryptotypes "github.com/stratosnet/sds/framework/crypto/types"

	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/tx-client/grpc"
	"github.com/stratosnet/sds/tx-client/types"
	authsigning "github.com/stratosnet/sds/tx-client/types/auth/signing"
	authtx "github.com/stratosnet/sds/tx-client/types/auth/tx"
	"github.com/stratosnet/sds/tx-client/types/tx/signing"
)

func CreateAndSimulateMultiMsgTx(msgs []*anypb.Any, txFee types.TxFee, memo string,
	signatureKeys []*types.SignatureKey, chainId string, gasAdjustment float64) ([]byte, error) {

	txConfig, unsignedTx := CreateTxConfigAndTxBuilder()
	setMsgInfosToTxBuilder(unsignedTx, msgs, txFee.Fee, txFee.Gas, memo)

	unsignedMsgs := make([]*types.UnsignedMsg, 0)
	for _, msg := range msgs {
		unsignedMsgs = append(unsignedMsgs, &types.UnsignedMsg{
			Msg: msg, SignatureKeys: signatureKeys, Type: msg.TypeUrl,
		})
	}

	txBytes, err := BuildTxBytes(txConfig, unsignedTx, chainId, unsignedMsgs)
	if err != nil {
		return nil, err
	}

	if txFee.Simulate {
		gasInfo, err := grpc.Simulate(txBytes)
		if err != nil {
			return nil, errors.Wrap(fmt.Errorf("failed to get gasInfo from chain"), err.Error())
		}
		unsignedTx.AuthInfo.Fee.GasLimit = uint64(float64(gasInfo.GasUsed) * gasAdjustment)

		txBytes, err = BuildTxBytes(txConfig, unsignedTx, chainId, unsignedMsgs)
		if err != nil {
			return nil, errors.Wrap(fmt.Errorf("failed to build txBytes"), err.Error())
		}
	}
	return txBytes, nil

}

func setMsgInfosToTxBuilder(unsignedTx *txv1beta1.Tx, txMsgs []*anypb.Any, fee types.Coin, gas uint64, memo string) {
	unsignedTx.Body = &txv1beta1.TxBody{
		Messages: txMsgs,
		Memo:     memo,
	}
	unsignedTx.AuthInfo = &txv1beta1.AuthInfo{
		Fee: &txv1beta1.Fee{
			Amount: []*basev1beta1.Coin{
				{
					Denom:  fee.Denom,
					Amount: fee.Amount.String(),
				},
			},
			GasLimit: gas,
		},
	}
	return
}

func CreateAndSimulateTx(msg *anypb.Any, txFee types.TxFee, memo string,
	signatureKeys []*types.SignatureKey, chainId string, gasAdjustment float64) ([]byte, error) {

	txConfig, unsignedTx := CreateTxConfigAndTxBuilder()
	setMsgInfoToTxBuilder(unsignedTx, msg, txFee.Fee, txFee.Gas, memo)

	unsignedMsgs := []*types.UnsignedMsg{{Msg: msg, SignatureKeys: signatureKeys, Type: msg.TypeUrl}}
	txBytes, err := BuildTxBytes(txConfig, unsignedTx, chainId, unsignedMsgs)
	if err != nil {
		return nil, err
	}

	if txFee.Simulate {
		gasInfo, err := grpc.Simulate(txBytes)
		if err != nil {
			return nil, errors.Wrap(fmt.Errorf("failed to get gasInfo from chain"), err.Error())
		}
		unsignedTx.AuthInfo.Fee.GasLimit = uint64(float64(gasInfo.GasUsed) * gasAdjustment)

		txBytes, err = BuildTxBytes(txConfig, unsignedTx, chainId, unsignedMsgs)
		if err != nil {
			return nil, errors.Wrap(fmt.Errorf("failed to build txBytes"), err.Error())
		}
	}
	return txBytes, nil

}

func setMsgInfoToTxBuilder(unsignedTx *txv1beta1.Tx, txMsg *anypb.Any, fee types.Coin, gas uint64, memo string) {
	unsignedTx.Body = &txv1beta1.TxBody{
		Messages: []*anypb.Any{txMsg},
		Memo:     memo,
	}
	unsignedTx.AuthInfo = &txv1beta1.AuthInfo{
		Fee: &txv1beta1.Fee{
			Amount: []*basev1beta1.Coin{
				{
					Denom:  fee.Denom,
					Amount: fee.Amount.String(),
				},
			},
			GasLimit: gas,
		},
	}
	return
}

func BuildTxBytes(txConfig TxConfig, unsignedTx *txv1beta1.Tx, chainId string, unsignedMsgs []*types.UnsignedMsg) ([]byte, error) {
	filteredMsgs := filterInvalidSignatures(unsignedMsgs)          // Filter msgs with invalid signatures
	accountInfos := fetchAllAccountInfos(filteredMsgs)             // Fetch account info from stratos-chain for each signature
	updatedMsgs := updateSignatureKeys(filteredMsgs, accountInfos) // Update signatureKeys for each msg

	if len(updatedMsgs) != len(unsignedMsgs) {
		utils.ErrorLogf("BuildTxBytes couldn't build all the msgs provided (success: %v  invalid_signature: %v  missing_account_infos: %v",
			len(updatedMsgs), len(unsignedMsgs)-len(filteredMsgs), len(filteredMsgs)-len(updatedMsgs))
	}

	if len(updatedMsgs) == 0 {
		return []byte{}, fmt.Errorf("no available account to sign transaction")
	}

	return buildAndSignStdTx(txConfig, unsignedTx, chainId, updatedMsgs)
}

func buildAndSignStdTx(txConfig TxConfig, unsignedTx *txv1beta1.Tx, chainId string, unsignedMsgs []*types.UnsignedMsg) ([]byte, error) {
	if len(unsignedMsgs) == 0 {
		return nil, fmt.Errorf("cannot build tx: no msgs to sign")
	}
	// Collect list of signatures to do. Must match order of GetSigners() method in cosmos-sdk's stdtx.go
	var signaturesToDo []*types.SignatureKey
	signersSeen := make(map[string]bool)
	for _, msg := range unsignedMsgs {
		for _, signaturekey := range msg.SignatureKeys {
			if !signersSeen[signaturekey.Address] {
				signersSeen[signaturekey.Address] = true
				signaturesToDo = append(signaturesToDo, signaturekey)
			}
		}
	}

	var sigsV2 []signing.SignatureV2
	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	for _, signatureKey := range signaturesToDo {
		var privKey fwcryptotypes.PrivKey
		var err error
		switch signatureKey.Type {
		case types.SignatureEd25519:
			if len(signatureKey.PrivateKey) != ed25519crypto.PrivateKeySize {
				return []byte{}, fmt.Errorf("ed25519 private key has wrong length: " + hex.EncodeToString(signatureKey.PrivateKey))
			}
			privKey = &ed25519.PrivKey{Key: signatureKey.PrivateKey}
		default:
			privKey = secp256k1.Generate(signatureKey.PrivateKey)
		}

		pubKeyAny, err := getPackedPubKeyAnyByPrivKey(privKey)
		if err != nil {
			return nil, err
		}

		sigV2 := signing.SignatureV2{
			PubKey: pubKeyAny,
			Data: &signing.SingleSignatureData{
				SignMode:  txConfig.SignModeHandler().DefaultMode(),
				Signature: nil,
			},
			Sequence: signatureKey.AccountSequence,
		}

		sigsV2 = append(sigsV2, sigV2)

		unsignedTx, err = SetSignatures(unsignedTx, sigsV2...)
		if err != nil {
			return []byte{}, err
		}
	}

	var signedTx *txv1beta1.Tx
	// Second round: all signer infos are set, so each signer can sign.
	for _, signatureKey := range signaturesToDo {
		signerData := authsigning.SignerData{
			ChainID:       chainId,
			AccountNumber: signatureKey.AccountNum,
			Sequence:      signatureKey.AccountSequence,
		}

		var privKey fwcryptotypes.PrivKey
		switch signatureKey.Type {
		case types.SignatureEd25519:
			privKey = &ed25519.PrivKey{Key: signatureKey.PrivateKey}
		default:
			privKey = secp256k1.Generate(signatureKey.PrivateKey)
		}
		sigV2, err := SignWithPrivKey(
			txConfig.SignModeHandler().DefaultMode(), signerData,
			unsignedTx, privKey, txConfig, signerData.Sequence)
		if err != nil {
			return []byte{}, err
		}

		signedTx, err = SetSignatures(unsignedTx, sigV2)
		if err != nil {
			return []byte{}, err
		}
	}

	txBytes, err := proto.Marshal(signedTx)
	if err != nil {
		return []byte{}, err
	}

	return txBytes, nil
}

func SetSignatures(tx *txv1beta1.Tx, signatures ...signing.SignatureV2) (*txv1beta1.Tx, error) {
	n := len(signatures)
	signerInfos := make([]*txv1beta1.SignerInfo, n)
	rawSigs := make([][]byte, n)

	for i, sig := range signatures {
		var modeInfo *txv1beta1.ModeInfo
		modeInfo, rawSigs[i] = authtx.SignatureDataToModeInfoAndSig(sig.Data)

		signerInfos[i] = &txv1beta1.SignerInfo{
			PublicKey: sig.PubKey,
			ModeInfo:  modeInfo,
			Sequence:  sig.Sequence,
		}
	}

	tx.AuthInfo.SignerInfos = signerInfos
	tx.Signatures = rawSigs

	return tx, nil
}

func filterInvalidSignatures(msgs []*types.UnsignedMsg) []*types.UnsignedMsg {
	var filteredMsgs []*types.UnsignedMsg
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

func fetchAllAccountInfos(msgs []*types.UnsignedMsg) map[string]*authv1beta1.BaseAccount {
	// Gather all accounts to fetch
	accountsToFetch := make(map[string]bool)
	for _, msg := range msgs {
		for _, signatureKey := range msg.SignatureKeys {
			accountsToFetch[signatureKey.Address] = true
		}
	}

	// Fetch all accounts in parallel
	results := make(map[string]*authv1beta1.BaseAccount)
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

func updateSignatureKeys(msgs []*types.UnsignedMsg, accountInfos map[string]*authv1beta1.BaseAccount) []*types.UnsignedMsg {
	var filteredMsgs []*types.UnsignedMsg
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
