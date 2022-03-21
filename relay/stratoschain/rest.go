package stratoschain

import (
	"bytes"
	ed25519crypto "crypto/ed25519"
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdkrest "github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/cosmos/cosmos-sdk/x/auth/client/rest"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/relay"
	"github.com/stratosnet/sds/relay/stratoschain/handlers"
	pottypes "github.com/stratosnet/stratos-chain/x/pot/types"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"
	"github.com/tendermint/tendermint/crypto"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/stratosnet/sds/pp/types"
	_ "github.com/stratosnet/sds/relay/stratoschain/prefix"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
)

var Url string

const (
	SignatureSecp256k1 = iota
	SignatureEd25519
)

type SignatureKey struct {
	AccountNum      uint64
	AccountSequence uint64
	Address         string
	PrivateKey      []byte
	Type            int
}

type UnsignedMsg struct {
	Msg           sdktypes.Msg
	SignatureKeys []SignatureKey
}

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

	var wrappedResponse sdkrest.ResponseWithHeight
	err = codec.Cdc.UnmarshalJSON(respBytes, &wrappedResponse)
	if err != nil {
		return nil, err
	}

	var account authtypes.BaseAccount
	err = authtypes.ModuleCdc.UnmarshalJSON(wrappedResponse.Result, &account)
	return &account, err
}

func buildAndSignStdTx(token, chainId, memo string, unsignedMsgs []*UnsignedMsg, fee, gas int64) (*authtypes.StdTx, error) {
	stdFee := authtypes.NewStdFee(
		uint64(gas),
		sdktypes.NewCoins(sdktypes.NewInt64Coin(token, fee)),
	)

	// Collect list of signatures to do. Must match order of GetSigners() method in cosmos-sdk's stdtx.go
	var signaturesToDo []SignatureKey
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
		case SignatureEd25519:
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

func BuildTxBytes(token, chainId, memo, mode string, unsignedMsgs []*UnsignedMsg, fee, gas int64) ([]byte, error) {
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

	body := rest.BroadcastReq{
		Tx:   *tx,
		Mode: mode,
	}

	return relay.Cdc.MarshalJSON(body)
}

func filterInvalidSignatures(msgs []*UnsignedMsg) []*UnsignedMsg {
	var filteredMsgs []*UnsignedMsg
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

func fetchAllAccountInfos(msgs []*UnsignedMsg) map[string]*authtypes.BaseAccount {
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

func updateSignatureKeys(msgs []*UnsignedMsg, accountInfos map[string]*authtypes.BaseAccount) []*UnsignedMsg {
	var filteredMsgs []*UnsignedMsg
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
		// TODO: QB-1064 use events from the result instead of unmarshalling the tx bytes
		var txResponse sdktypes.TxResponse

		err = relay.Cdc.UnmarshalJSON(responseBody, &txResponse)
		if err != nil {
			return errors.Wrap(err, "couldn't unmarshal response body to txResponse")
		}

		if txResponse.Height <= 0 || txResponse.Empty() || txResponse.Code != 0 {
			return errors.Errorf("broadcast unsuccessful: %v", txResponse)
		}

		if len(broadcastReq.Tx.Msgs) < 1 {
			return errors.New("broadcastReq tx doesn't contain any messages")
		}

		// Additional processing based on the msg type
		slashPPEvents := make(map[string][]string)
		for _, msg := range broadcastReq.Tx.Msgs {
			if msg.Type() == "slashing_resource_node" {
				slashMsg, ok := msg.(pottypes.MsgSlashingResourceNode)
				if !ok {
					return errors.New("cannot convert msg to MsgSlashingResourceNode")
				}

				slashPPEvents["slashing.network_address"] = append(slashPPEvents["slashing.network_address"], slashMsg.NetworkAddress.String())
				slashPPEvents["slashing.suspended"] = append(slashPPEvents["slashing.suspended"], strconv.FormatBool(slashMsg.Suspend))
			}
		}
		if len(slashPPEvents) > 0 {
			// Directly call slashing_resource_node handler
			result := coretypes.ResultEvent{Events: slashPPEvents}
			handlers.SlashingResourceNodeHandler()(result)
		}
	}

	return nil
}

type ResourceNodeState struct {
	IsActive  uint32
	Suspended bool
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
	return state, nil
}
