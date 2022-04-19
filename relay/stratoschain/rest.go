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

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdkrest "github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/cosmos/cosmos-sdk/x/auth/client/rest"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/pkg/errors"
	setting "github.com/stratosnet/sds/cmd/relayd/config"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/relay"
	"github.com/stratosnet/sds/relay/stratoschain/handlers"
	_ "github.com/stratosnet/sds/relay/stratoschain/prefix"
	relaytypes "github.com/stratosnet/sds/relay/types"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"
	"github.com/tendermint/tendermint/crypto"
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

	var wrappedResponse sdkrest.ResponseWithHeight
	err = codec.Cdc.UnmarshalJSON(respBytes, &wrappedResponse)
	if err != nil {
		return nil, err
	}

	var account authtypes.BaseAccount
	err = authtypes.ModuleCdc.UnmarshalJSON(wrappedResponse.Result, &account)
	return &account, err
}

func buildAndSignStdTx(token, chainId, memo string, unsignedMsgs []*relaytypes.UnsignedMsg, fee, gas int64) (*authtypes.StdTx, error) {
	if len(unsignedMsgs) == 0 {
		return nil, errors.New("cannot build tx: no msgs to sign")
	}

	stdFee := authtypes.NewStdFee(
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
	var signatures []authtypes.StdSignature
	for _, signatureKey := range signaturesToDo {
		unsignedBytes := authtypes.StdSignBytes(chainId, signatureKey.AccountNum, signatureKey.AccountSequence, stdFee, sdkMsgs, memo)

		var signedBytes []byte
		var pubKey crypto.PubKey

		switch signatureKey.Type {
		case relaytypes.SignatureEd25519:
			if len(signatureKey.PrivateKey) != ed25519crypto.PrivateKeySize {
				return nil, errors.New("ed25519 private key has wrong length: " + hex.EncodeToString(signatureKey.PrivateKey))
			}

			signedBytes = ed25519crypto.Sign(signatureKey.PrivateKey, unsignedBytes)
			pubKey = ed25519.PrivKeyBytesToPubKey(signatureKey.PrivateKey)
		default:
			var err error

			signedBytes, err = secp256k1.PrivKeyBytesToTendermint(signatureKey.PrivateKey).Sign(unsignedBytes)
			if err != nil {
				return nil, err
			}

			pubKey, err = secp256k1.PubKeyBytesToTendermint(secp256k1.PrivKeyToPubKey(signatureKey.PrivateKey))
			if err != nil {
				return nil, err
			}
		}

		sig := authtypes.StdSignature{
			PubKey:    pubKey,
			Signature: signedBytes,
		}
		signatures = append(signatures, sig)
	}

	tx := authtypes.NewStdTx(sdkMsgs, stdFee, signatures, memo)
	return &tx, nil
}

func BuildTxBytes(token, chainId, memo, mode string, unsignedMsgs []*relaytypes.UnsignedMsg, fee, gas int64) ([]byte, error) {
	filteredMsgs := filterInvalidSignatures(unsignedMsgs)          // Filter msgs with invalid signatures
	accountInfos := fetchAllAccountInfos(filteredMsgs)             // Fetch account info from stratos-chain for each signature
	updatedMsgs := updateSignatureKeys(filteredMsgs, accountInfos) // Update signatureKeys for each msg

	tx, err := buildAndSignStdTx(token, chainId, memo, updatedMsgs, fee, gas)
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

	body := rest.BroadcastReq{
		Tx:   *tx,
		Mode: mode,
	}

	return relay.Cdc.MarshalJSON(body)
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

	url, err := utils.ParseUrl(Url + "/txs")
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

	var broadcastReq rest.BroadcastReq
	err = relay.Cdc.UnmarshalJSON(txBytes, &broadcastReq)
	if err != nil {
		return errors.Wrap(err, "cannot unmarshal txBytes to BroadcastReq")
	}

	// In block broadcast mode, do additional verification
	if broadcastReq.Mode == flags.BroadcastBlock {
		var txResponse sdktypes.TxResponse

		err = relay.Cdc.UnmarshalJSON(responseBody, &txResponse)
		if err != nil {
			return errors.Wrap(err, "couldn't unmarshal response body to txResponse")
		}

		if txResponse.Height <= 0 || txResponse.Empty() || txResponse.Code != 0 {
			return errors.Errorf("broadcast unsuccessful: %v", txResponse)
		}

		if setting.Config == nil {
			return nil // If the relayd config is nil, then this is ppd broadcasting a tx. We don't want to call the event handler in this case
		}
		events := processEvents(txResponse)
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

func processEvents(response sdktypes.TxResponse) map[string]coretypes.ResultEvent {
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

func QueryResourceNodeState(p2pAddress string) (state ResourceNodeState, err error) {
	state = ResourceNodeState{
		IsActive:  types.PP_INACTIVE,
		Suspended: true,
	}
	if Url == "" {
		return state, errors.New("the stratos-chain URL is not set")
	}

	url, err := utils.ParseUrl(Url + "/register/resource-nodes?network=" + p2pAddress)
	if err != nil {
		return state, err
	}

	resp, err := http.Get(url.String(true, true, true, true))
	if err != nil {
		return state, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return state, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return state, nil
	}

	if resp.StatusCode != http.StatusOK {
		return state, errors.Errorf("HTTP%v: %v", resp.StatusCode, string(respBody))
	}

	var wrappedResponse sdkrest.ResponseWithHeight
	err = codec.Cdc.UnmarshalJSON(respBody, &wrappedResponse)
	if err != nil {
		return state, err
	}

	var resourceNodes registertypes.ResourceNodes
	err = authtypes.ModuleCdc.UnmarshalJSON(wrappedResponse.Result, &resourceNodes)
	if err != nil {
		return state, err
	}

	if len(resourceNodes) == 0 {
		return state, nil
	}
	if resourceNodes[0].GetNetworkAddr().String() != p2pAddress {
		return state, nil
	}

	state.Suspended = resourceNodes[0].IsSuspended()
	switch resourceNodes[0].GetStatus() {
	case sdktypes.Bonded:
		state.IsActive = types.PP_ACTIVE
	case sdktypes.Unbonding:
		state.IsActive = types.PP_UNBONDING
	case sdktypes.Unbonded:
		state.IsActive = types.PP_INACTIVE
	}

	state.Tokens = resourceNodes[0].GetTokens().BigInt()
	return state, nil
}
