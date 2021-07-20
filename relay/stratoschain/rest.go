package stratoschain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdkrest "github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/cosmos/cosmos-sdk/x/auth/client/rest"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	pottypes "github.com/stratosnet/stratos-chain/x/pot/types"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"
	sdstypes "github.com/stratosnet/stratos-chain/x/sds/types"
	"io/ioutil"
	"net/http"
)

var Url string
var cdc *codec.Codec

func init() {
	cdc = codec.New()
	codec.RegisterCrypto(cdc)
	sdktypes.RegisterCodec(cdc)
	registertypes.RegisterCodec(cdc)
	sdstypes.RegisterCodec(cdc)
	pottypes.RegisterCodec(cdc)
	cdc.Seal()
}

func FetchAccountInfo(address string) (uint64, uint64, error) {
	if Url == "" {
		return 0, 0, errors.New("the stratos-chain URL is not set")
	}

	resp, err := http.Get(Url + "/auth/accounts/" + address)
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

func BuildAndSignTx(token, chainId, memo string, accountNum, sequence uint64, msg sdktypes.Msg, fee, gas int64, privateKey []byte) (*authtypes.StdTx, error) {
	stdFee := authtypes.NewStdFee(
		uint64(gas),
		sdktypes.NewCoins(sdktypes.NewInt64Coin(token, fee)),
	)
	msgs := []sdktypes.Msg{msg}

	unsignedBytes := authtypes.StdSignBytes(chainId, accountNum, sequence, stdFee, msgs, memo)
	signedBytes, err := secp256k1.PrivKeyBytesToTendermint(privateKey).Sign(unsignedBytes)
	if err != nil {
		return nil, err
	}

	pubKey, err := secp256k1.PubKeyBytesToTendermint(secp256k1.PrivKeyToPubKey(privateKey))
	if err != nil {
		return nil, err
	}
	sig := authtypes.StdSignature{
		PubKey:    pubKey,
		Signature: signedBytes,
	}

	tx := authtypes.NewStdTx(msgs, stdFee, []authtypes.StdSignature{sig}, memo)
	return &tx, nil
}

func BuildTxBytes(token, chainId, memo, address, mode string, msg sdktypes.Msg, fee, gas int64, privateKey []byte) ([]byte, error) {
	accountNum, sequence, err := FetchAccountInfo(address)
	if err != nil {
		return nil, err
	}

	tx, err := BuildAndSignTx(token, chainId, memo, accountNum, sequence, msg, fee, gas, privateKey)
	if err != nil {
		return nil, err
	}

	body := rest.BroadcastReq{
		Tx:   *tx,
		Mode: mode,
	}

	return cdc.MarshalJSON(body)
}

func BroadcastTx(tx authtypes.StdTx) (*http.Response, []byte, error) {
	if Url == "" {
		return nil, nil, errors.New("the stratos-chain URL is not set")
	}

	body := rest.BroadcastReq{
		Tx:   tx,
		Mode: "sync",
	}
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return nil, nil, err
	}

	bodyBytes := bytes.NewBuffer(jsonBytes)
	resp, err := http.Post(Url+"/txs", "application/json", bodyBytes)
	if err != nil {
		return resp, nil, err
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	return resp, responseBody, err
}

func BroadcastTxBytes(txBytes []byte) error {
	if Url == "" {
		return errors.New("the stratos-chain URL is not set")
	}

	bodyBytes := bytes.NewBuffer(txBytes)
	resp, err := http.Post(Url+"/txs", "application/json", bodyBytes)
	if err != nil {
		return err
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(responseBody))

	if resp.StatusCode != 200 {
		return errors.New("invalid http response: " + resp.Status)
	}

	return err
}
