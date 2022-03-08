package stratoschain

import (
	"bytes"
	ed25519crypto "crypto/ed25519"
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"strconv"

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

func FetchAccountInfo(address string) (uint64, uint64, error) {
	if Url == "" {
		return 0, 0, errors.New("the stratos-chain URL is not set")
	}

	url, err := utils.ParseUrl(Url + "/auth/accounts/" + address)
	if err != nil {
		return 0, 0, err
	}
	resp, err := http.Get(url.String(true, true, true, false))
	if err != nil {
		return 0, 0, err
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, err
	}

	var wrappedResponse sdkrest.ResponseWithHeight
	err = codec.Cdc.UnmarshalJSON(respBytes, &wrappedResponse)
	if err != nil {
		return 0, 0, err
	}

	var account authtypes.BaseAccount
	err = authtypes.ModuleCdc.UnmarshalJSON(wrappedResponse.Result, &account)
	return account.AccountNumber, account.Sequence, err
}

func BuildAndSignTx(token, chainId, memo string, msg sdktypes.Msg, fee, gas int64, signatureKeys []SignatureKey) (*authtypes.StdTx, error) {
	stdFee := authtypes.NewStdFee(
		uint64(gas),
		sdktypes.NewCoins(sdktypes.NewInt64Coin(token, fee)),
	)
	msgs := []sdktypes.Msg{msg}

	var signatures []authtypes.StdSignature
	for _, signatureKey := range signatureKeys {
		unsignedBytes := authtypes.StdSignBytes(chainId, signatureKey.AccountNum, signatureKey.AccountSequence, stdFee, msgs, memo)

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

	tx := authtypes.NewStdTx(msgs, stdFee, signatures, memo)
	return &tx, nil
}

func BuildTxBytes(token, chainId, memo, mode string, msg sdktypes.Msg, fee, gas int64, signatureKeys []SignatureKey) ([]byte, error) {
	// Fetch account info from stratos-chain for each signature
	for i, signatureKey := range signatureKeys {
		if len(signatureKey.Address) == 0 {
			utils.ErrorLog("Wallet address is empty, failed to build Tx bytes.")
		}
		accountNum, sequence, err := FetchAccountInfo(signatureKey.Address)
		if err != nil {
			return nil, err
		}
		signatureKeys[i].AccountNum = accountNum
		signatureKeys[i].AccountSequence = sequence
	}

	tx, err := BuildAndSignTx(token, chainId, memo, msg, fee, gas, signatureKeys)
	if err != nil {
		return nil, err
	}

	body := rest.BroadcastReq{
		Tx:   *tx,
		Mode: mode,
	}

	return relay.Cdc.MarshalJSON(body)
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
		return errors.Errorf("invalid response HTTP%v: %v", resp.Status, responseBody)
	}

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

		if len(broadcastReq.Tx.Msgs) < 1 {
			return errors.New("broadcastReq tx doesn't contain any messages")
		}

		msg := broadcastReq.Tx.Msgs[0]
		if msg.Type() == "slashing_resource_node" {
			// Directly call slashing_resource_node handler
			slashMsg, ok := msg.(pottypes.MsgSlashingResourceNode)
			if !ok {
				return errors.New("cannot convert msg to MsgSlashingResourceNode")
			}

			result := coretypes.ResultEvent{Events: make(map[string][]string, 0)}
			result.Events["slashing_resource_node.network_address"] = []string{slashMsg.NetworkAddress.String()}
			result.Events["slashing_resource_node.suspended"] = []string{strconv.FormatBool(slashMsg.Suspend)}
			handlers.SlashingResourceNodeHandler()(result)
		}
	}

	return nil
}

func QueryResourceNodeState(networkId string) (int, error) {
	if Url == "" {
		return 0, errors.New("the stratos-chain URL is not set")
	}

	url, err := utils.ParseUrl(Url + "/register/resource-nodes?network=" + networkId)
	if err != nil {
		return 0, err
	}

	resp, err := http.Get(url.String(true, true, true, true))
	if err != nil {
		return 0, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var wrappedResponse sdkrest.ResponseWithHeight
	err = codec.Cdc.UnmarshalJSON(respBody, &wrappedResponse)
	if err != nil {
		return 0, err
	}

	var resourceNodes registertypes.ResourceNodes
	err = authtypes.ModuleCdc.UnmarshalJSON(wrappedResponse.Result, &resourceNodes)
	if err != nil {
		return 0, err
	}

	if len(resourceNodes) == 0 {
		return types.PP_INACTIVE, nil
	}
	if resourceNodes[0].IsSuspended() {
		return types.PP_SUSPENDED, nil
	}
	if resourceNodes[0].GetStatus() == sdktypes.Unbonding {
		return types.PP_UNBONDING, nil
	}
	if resourceNodes[0].GetStatus() == sdktypes.Bonded && resourceNodes[0].GetNetworkID() == networkId {
		return types.PP_ACTIVE, nil
	}
	return types.PP_INACTIVE, nil
}
