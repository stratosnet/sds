package stratoschain

import (
	"bytes"
	ed25519crypto "crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/cosmos/cosmos-sdk/codec"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdkrest "github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/cosmos/cosmos-sdk/x/auth/client/rest"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	pottypes "github.com/stratosnet/stratos-chain/x/pot/types"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"
	sdstypes "github.com/stratosnet/stratos-chain/x/sds/types"
	"github.com/tendermint/tendermint/crypto"
	"io/ioutil"
	"net/http"
)

var Url string
var Cdc *codec.Codec

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

func init() {
	Cdc = codec.New()
	codec.RegisterCrypto(Cdc)
	sdktypes.RegisterCodec(Cdc)
	registertypes.RegisterCodec(Cdc)
	sdstypes.RegisterCodec(Cdc)
	pottypes.RegisterCodec(Cdc)
	Cdc.Seal()
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

	return Cdc.MarshalJSON(body)
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
	utils.Log(string(responseBody))

	if resp.StatusCode != 200 {
		return errors.New("invalid http response: " + resp.Status)
	}

	return err
}
