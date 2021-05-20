package stratoschain

import (
	"bytes"
	"encoding/json"
	"errors"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdkrest "github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/cosmos/cosmos-sdk/x/auth/client/rest"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	"github.com/stratosnet/sds/utils/types"
	"io/ioutil"
	"net/http"
)

var Url string

func FetchAccountInfo(address types.Address, bechPrefix string) (uint64, uint64, error) {
	if Url == "" {
		return 0, 0, errors.New("the stratos-chain URL is not set")
	}

	bechAddress, err := address.ToBech(bechPrefix)
	if err != nil {
		return 0, 0, err
	}

	resp, err := http.Get(Url + "/auth/accounts/" + bechAddress)
	if err != nil {
		return 0, 0, err
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, err
	}

	var wrappedResponse sdkrest.ResponseWithHeight
	err = json.Unmarshal(respBytes, &wrappedResponse)
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
	signedBytes, err := secp256k1.Sign(unsignedBytes, privateKey)
	if err != nil {
		return nil, err
	}

	sig := authtypes.StdSignature{Signature: signedBytes}

	tx := authtypes.NewStdTx(msgs, stdFee, []authtypes.StdSignature{sig}, memo)
	return &tx, nil
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
