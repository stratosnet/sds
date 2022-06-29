package stratoschain

import (
	"bytes"
	ed25519crypto "crypto/ed25519"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"sync"

	"github.com/cosmos/cosmos-sdk/client"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdkrest "github.com/cosmos/cosmos-sdk/types/rest"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"

	//txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/auth/legacy/legacytx"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/pkg/errors"
	setting "github.com/stratosnet/sds/cmd/relayd/config"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/relay"
	"github.com/stratosnet/sds/relay/stratoschain/handlers"
	_ "github.com/stratosnet/sds/relay/stratoschain/prefix"
	relaytypes "github.com/stratosnet/sds/relay/types"
	"github.com/stratosnet/sds/utils"
	//"github.com/stratosnet/sds/utils/crypto"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	//"github.com/ethereum/go-ethereum/crypto/secp256k1"
	utilsecp256k1 "github.com/stratosnet/sds/utils/crypto/secp256k1"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"
	//"github.com/tendermint/tendermint/crypto"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

var Url string

func FetchAccountInfo(address string) (*authtypes.BaseAccount, error) {
	if Url == "" {
		return nil, errors.New("the stratos-chain URL is not set")
	}

	url, err := utils.ParseUrl(Url + "/auth/accounts/" + address)
	if err != nil {
		return nil, err
	}
	resp, err := http.Get(url.String(true, true, true, false))
	if err != nil {
		return nil, err
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, errors.Errorf("invalid response HTTP%v: %v", resp.Status, string(respBytes))
	}

	responseResult, err := sdkrest.ParseResponseWithHeight(relay.Cdc, respBytes)
	//var wrappedResponse sdkrest.ResponseWithHeight
	//err = codec.Codec.UnmarshalJSON(respBytes, &wrappedResponse)
	if err != nil {
		return nil, err
	}

	//var account authtypes.BaseAccount
	var account authtypes.BaseAccount
	err = authtypes.ModuleCdc.UnmarshalJSON(responseResult, &account)
	//err = account.Unmarshal(responseResult)
	return &account, err
}

func buildAndSignStdTxNew(protoConfig client.TxConfig, txBuilder client.TxBuilder, token, chainId, memo string, unsignedMsgs []*relaytypes.UnsignedMsg, fee, gas, height int64) ([]byte, error) {
	if len(unsignedMsgs) == 0 {
		return nil, errors.New("cannot build tx: no msgs to sign")
	}
	// Collect list of signatures to do. Must match order of GetSigners() method in cosmos-sdk's stdtx.go
	var signaturesToDo []relaytypes.SignatureKey
	signersSeen := make(map[string]bool)
	var sdkMsgs []sdktypes.Msg
	for _, msg := range unsignedMsgs {
		for _, signaturekey := range msg.SignatureKeys {
			if !signersSeen[signaturekey.Address] {
				signersSeen[signaturekey.Address] = true
				signaturesToDo = append(signaturesToDo, signaturekey)
			}
		}
		sdkMsgs = append(sdkMsgs, msg.Msg)
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
			pubkey = utilsecp256k1.PrivKeyBytesToSdkPriv(signatureKey.PrivateKey).PubKey()
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
			privKey = utilsecp256k1.PrivKeyBytesToSdkPriv(signatureKey.PrivateKey)
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

	txReq := &sdktx.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    sdktx.BroadcastMode_BROADCAST_MODE_BLOCK,
	}
	reqBytes, _ := relay.ProtoCdc.MarshalJSON(txReq)
	return reqBytes, nil
}

func BuildTxBytesNew(protoConfig client.TxConfig, txBuilder client.TxBuilder, token, chainId, memo, mode string, unsignedMsgs []*relaytypes.UnsignedMsg, fee, gas, height int64) ([]byte, error) {
	filteredMsgs := filterInvalidSignatures(unsignedMsgs)          // Filter msgs with invalid signatures
	accountInfos := fetchAllAccountInfos(filteredMsgs)             // Fetch account info from stratos-chain for each signature
	updatedMsgs := updateSignatureKeys(filteredMsgs, accountInfos) // Update signatureKeys for each msg

	if len(updatedMsgs) != len(unsignedMsgs) {
		utils.ErrorLogf("BuildTxBytes couldn't build all the msgs provided (success: %v  invalid_signature: %v  missing_account_infos: %v",
			len(updatedMsgs), len(unsignedMsgs)-len(filteredMsgs), len(filteredMsgs)-len(updatedMsgs))
	}

	return buildAndSignStdTxNew(protoConfig, txBuilder, token, chainId, memo, updatedMsgs, fee, gas, height)
}

func buildAndSignStdTx(token, chainId, memo string, unsignedMsgs []*relaytypes.UnsignedMsg, fee, gas, height int64) (*legacytx.StdTx, error) {
	if len(unsignedMsgs) == 0 {
		return nil, errors.New("cannot build tx: no msgs to sign")
	}

	tx := legacytx.StdTx{TimeoutHeight: uint64(height) + relaytypes.DefaultTimeoutHeight}

	stdFee := legacytx.NewStdFee(
		uint64(gas),
		sdktypes.NewCoins(sdktypes.NewInt64Coin(token, fee)),
	)

	// Collect list of signatures to do. Must match order of GetSigners() method in cosmos-sdk's stdtx.go
	var signaturesToDo []relaytypes.SignatureKey
	signersSeen := make(map[string]bool)
	var sdkMsgs []sdktypes.Msg
	for _, msg := range unsignedMsgs {
		for _, signaturekey := range msg.SignatureKeys {
			if !signersSeen[signaturekey.Address] {
				signersSeen[signaturekey.Address] = true
				signaturesToDo = append(signaturesToDo, signaturekey)
			}
		}
		sdkMsgs = append(sdkMsgs, msg.Msg)
	}

	// Sign the tx
	var signatures []legacytx.StdSignature
	for _, signatureKey := range signaturesToDo {
		unsignedBytes := legacytx.StdSignBytes(chainId, signatureKey.AccountNum, signatureKey.AccountSequence, tx.GetTimeoutHeight(), stdFee, sdkMsgs, memo)

		var signedBytes []byte
		var pubKey cryptotypes.PubKey

		switch signatureKey.Type {
		case relaytypes.SignatureEd25519:
			if len(signatureKey.PrivateKey) != ed25519crypto.PrivateKeySize {
				return nil, errors.New("ed25519 private key has wrong length: " + hex.EncodeToString(signatureKey.PrivateKey))
			}

			signedBytes = ed25519crypto.Sign(signatureKey.PrivateKey, unsignedBytes)
			pubKey = ed25519.PrivKeyBytesToSdkPubKey(signatureKey.PrivateKey)
		default:
			var err error

			signedBytes, err = utilsecp256k1.PrivKeyBytesToTendermint(signatureKey.PrivateKey).Sign(unsignedBytes)
			if err != nil {
				return nil, err
			}

			tmPubKey, err := utilsecp256k1.PubKeyBytesToTendermint(utilsecp256k1.PrivKeyToPubKey(signatureKey.PrivateKey))
			if err != nil {
				return nil, err
			}

			pubKey = utilsecp256k1.PubKeyBytesToSdkPubKey(tmPubKey.Bytes())
		}

		sig := legacytx.StdSignature{
			PubKey:    pubKey,
			Signature: signedBytes,
		}
		signatures = append(signatures, sig)
	}

	tx.Msgs = sdkMsgs
	tx.Fee = stdFee
	tx.Signatures = signatures
	tx.Memo = memo
	//tx := legacytx.NewStdTx(sdkMsgs, stdFee, signatures, memo)
	txBytes, _ := relay.Cdc.MarshalJSON(tx.Msgs)
	txHexStr := hex.EncodeToString(txBytes)
	fmt.Print("tx hex str: " + txHexStr)
	return &tx, nil
}

func BuildTxBytes(token, chainId, memo, mode string, unsignedMsgs []*relaytypes.UnsignedMsg, fee, gas, height int64) ([]byte, error) {
	filteredMsgs := filterInvalidSignatures(unsignedMsgs)          // Filter msgs with invalid signatures
	accountInfos := fetchAllAccountInfos(filteredMsgs)             // Fetch account info from stratos-chain for each signature
	updatedMsgs := updateSignatureKeys(filteredMsgs, accountInfos) // Update signatureKeys for each msg

	tx, err := buildAndSignStdTx(token, chainId, memo, updatedMsgs, fee, gas, height)
	if err != nil {
		return nil, err
	}

	if len(updatedMsgs) != len(unsignedMsgs) {
		utils.ErrorLogf("BuildTxBytes couldn't build all the msgs provided (success: %v  invalid_signature: %v  missing_account_infos: %v",
			len(updatedMsgs), len(unsignedMsgs)-len(filteredMsgs), len(filteredMsgs)-len(updatedMsgs))
	}

	// Print account sequences
	accountsStr := ""
	for walletAddress, account := range accountInfos {
		if accountsStr != "" {
			accountsStr += ", "
		}
		accountsStr += fmt.Sprintf("(Wallet %v  Num %v  Sequence %v)", walletAddress, account.AccountNumber, account.Sequence)
	}
	utils.DebugLogf("BuildTxBytes ChainId [%v] Accounts [%v] Mode [%v]", chainId, accountsStr, mode)

	broadcastmode := sdktx.BroadcastMode_BROADCAST_MODE_BLOCK

	protoConfig := authtx.NewTxConfig(relay.ProtoCdc, []signingtypes.SignMode{signingtypes.SignMode_SIGN_MODE_DIRECT})
	txBuilder := protoConfig.NewTxBuilder()
	txBuilder.SetFeeAmount(tx.Fee.Amount)
	txBuilder.SetFeeGranter(tx.FeeGranter())
	txBuilder.SetGasLimit(tx.Fee.Gas)
	txBuilder.SetMemo(tx.Memo)
	txBuilder.SetMsgs(tx.Msgs...)
	//txBuilder.SetTimeoutHeight(tx.TimeoutHeight)
	txBuilder.SetSignatures(signingtypes.SignatureV2{
		PubKey: tx.Signatures[0].PubKey,
		Data: &signingtypes.SingleSignatureData{
			SignMode:  signingtypes.SignMode_SIGN_MODE_DIRECT,
			Signature: tx.Signatures[0].Signature,
		},
	})
	txBytes, _ := protoConfig.TxEncoder()(txBuilder.GetTx())
	txReq := &sdktx.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    broadcastmode,
	}
	return relay.ProtoCdc.MarshalJSON(txReq)
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

			baseAccount, err := FetchAccountInfo(walletAddress)
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
			signatureKey.AccountNum = info.AccountNumber
			signatureKey.AccountSequence = info.Sequence
			msg.SignatureKeys[i] = signatureKey
		}
		if missingInfos {
			continue
		}

		filteredMsgs = append(filteredMsgs, msg)
	}

	return filteredMsgs
}

func BroadcastTxBytes(txBytes []byte) error {
	if Url == "" {
		return errors.New("the stratos-chain URL is not set")
	}

	url, err := utils.ParseUrl(Url + "/cosmos/tx/v1beta1/txs")
	if err != nil {
		return err
	}

	bodyBytes := bytes.NewBuffer(txBytes)
	resp, err := http.Post(url.String(true, true, true, false), "application/json", bodyBytes)
	if err != nil {
		return err
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.Errorf("invalid response HTTP%v: %v", resp.Status, string(responseBody))
	}
	utils.Log(string(responseBody))

	//var broadcastReq tx.BroadcastTxRequest
	var broadcastReq cli.BroadcastReq
	err = relay.Cdc.UnmarshalJSON(txBytes, &broadcastReq)
	if err != nil {
		return errors.Wrap(err, "cannot unmarshal txBytes to BroadcastReq")
	}

	// In block broadcast mode, do additional verification
	if broadcastReq.Mode == sdktx.BroadcastMode_BROADCAST_MODE_BLOCK.String() {
		var broadcastTxResponse sdktx.BroadcastTxResponse
		//var txResponse sdktypes.TxResponse

		err = relay.ProtoCdc.UnmarshalJSON(responseBody, &broadcastTxResponse)
		if err != nil {
			return errors.Wrap(err, "couldn't unmarshal response body to txResponse")
		}
		txResponse := broadcastTxResponse.TxResponse
		if txResponse.Height <= 0 || txResponse.Empty() || txResponse.Code != 0 {
			return errors.Errorf("broadcast unsuccessful: %v", txResponse)
		}

		if setting.Config == nil {
			return nil // If the relayd config is nil, then this is ppd broadcasting a tx. We don't want to call the event handler in this case
		}
		events := processEvents(broadcastTxResponse)
		for msgType, event := range events {
			if handler, ok := handlers.Handlers[msgType]; ok {
				go handler(event)
			} else {
				utils.ErrorLogf("No handler for event type [%v]", msgType)
			}
		}
	}

	return nil
}

func processEvents(broadcastResponse sdktx.BroadcastTxResponse) map[string]coretypes.ResultEvent {
	response := broadcastResponse.TxResponse
	// Read the events from each msg in the log
	var events []map[string]string
	for _, msg := range response.Logs {
		msgMap := make(map[string]string)
		for _, stringEvent := range msg.Events {
			for _, attrib := range stringEvent.Attributes {
				msgMap[fmt.Sprintf("%v.%v", stringEvent.Type, attrib.Key)] = attrib.Value
			}
		}
		if len(msgMap) > 0 {
			events = append(events, msgMap)
		}
	}

	// Aggregate events by msg type
	aggregatedEvents := make(map[string]map[string][]string)
	for _, event := range events {
		typeStr := event["message.action"]
		currentMap := aggregatedEvents[typeStr]
		if currentMap == nil {
			currentMap = make(map[string][]string)
			currentMap["tx.hash"] = []string{response.TxHash}
		}

		for key, value := range event {
			switch key {
			case "message.action":
				continue
			default:
				currentMap[key] = append(currentMap[key], value)
			}
		}
		aggregatedEvents[typeStr] = currentMap
	}

	// Convert to coretypes.ResultEvent
	resultMap := make(map[string]coretypes.ResultEvent)
	for key, value := range aggregatedEvents {
		resultMap[key] = coretypes.ResultEvent{
			Query:  "",
			Data:   nil,
			Events: value,
		}
	}
	return resultMap
}

type ResourceNodeState struct {
	IsActive  uint32
	Suspended bool
	Tokens    *big.Int
}

func ParseResponseWithHeight(cdc *codec.LegacyAmino, bz []byte) ([]byte, int64, error) {
	r := sdkrest.ResponseWithHeight{}
	if err := cdc.UnmarshalJSON(bz, &r); err != nil {
		return nil, int64(0), err
	}

	return r.Result, r.Height, nil
}

func QueryResourceNodeState(p2pAddress string) (state ResourceNodeState, height int64, err error) {
	state = ResourceNodeState{
		IsActive:  types.PP_INACTIVE,
		Suspended: true,
	}
	if Url == "" {
		return state, int64(0), errors.New("the stratos-chain URL is not set")
	}

	url, err := utils.ParseUrl(Url + "/register/resource-node/" + p2pAddress)
	if err != nil {
		return state, int64(0), err
	}

	resp, err := http.Get(url.String(true, true, true, true))
	if err != nil {
		return state, int64(0), err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return state, int64(0), err
	}

	if resp.StatusCode == http.StatusNotFound {
		return state, int64(0), nil
	}

	if resp.StatusCode != http.StatusOK {
		return state, int64(0), errors.Errorf("HTTP%v: %v", resp.StatusCode, string(respBody))
	}

	responseResult, height, err := ParseResponseWithHeight(relay.Cdc, respBody)
	//var wrappedResponse sdkrest.ResponseWithHeight
	//err = codec.Cdc.UnmarshalJSON(respBody, &wrappedResponse)
	if err != nil {
		return state, height, err
	}

	var resourceNode registertypes.ResourceNode
	err = registertypes.ModuleCdc.UnmarshalJSON(responseResult, &resourceNode)
	if err != nil {
		return state, height, err
	}

	//if len(resourceNodes) == 0 {
	//	return state, height, nil
	//}
	if resourceNode.GetNetworkAddress() != p2pAddress {
		return state, height, nil
	}

	state.Suspended = resourceNode.Suspend
	switch resourceNode.GetStatus() {
	case stakingtypes.Bonded:
		state.IsActive = types.PP_ACTIVE
	case stakingtypes.Unbonding:
		state.IsActive = types.PP_UNBONDING
	case stakingtypes.Unbonded:
		state.IsActive = types.PP_INACTIVE
	}

	state.Tokens = resourceNode.Tokens.BigInt()
	return state, height, nil
}
