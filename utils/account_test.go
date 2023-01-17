package utils

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	utilsecp256k1 "github.com/stratosnet/sds/utils/crypto/secp256k1"
	"github.com/stratosnet/sds/utils/types"
	stchaintypes "github.com/stratosnet/stratos-chain/types"
	tmed25519 "github.com/tendermint/tendermint/crypto/ed25519"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/privval"
)

var cdc = codec.NewLegacyAmino()

func init() {
	cryptocodec.RegisterCrypto(cdc)
}

func TestCreateWallet(t *testing.T) {
	t.SkipNow() // Comment this line out to run the method and create wallets
	for i := 0; i < 1; i++ {
		password := "aaa"

		mnemonic, err := NewMnemonic()
		if err != nil {
			t.Fatal(err.Error())
		}
		addr, err := CreateWallet("keys", "", password, stchaintypes.StratosBech32Prefix, mnemonic, "", "m/44'/606'/0'/0/0")
		if err != nil {
			t.Fatal(err.Error())
		}

		bechAddr, err := addr.WalletAddressToBech()
		if err != nil {
			t.Fatal(err.Error())
		}

		keyjson, err := ioutil.ReadFile("keys/" + bechAddr + ".json")
		if err != nil {
			t.Fatal(err.Error())
		}
		key, err := DecryptKey(keyjson, password)
		if err != nil {
			t.Fatal(err.Error())
		}

		sdkPubkey := utilsecp256k1.PrivKeyToPubKey(key.PrivateKey)
		bechPub, err := bech32.ConvertAndEncode(stchaintypes.AccountPubKeyPrefix, sdkPubkey.Bytes())
		if err != nil {
			t.Fatal(err.Error())
		}
		fmt.Printf("Address: %v  PublicKey: %v  Mnemonic: %v\n", bechAddr, bechPub, key.Mnemonic)
	}
}

func TestCreateP2PKey(t *testing.T) {
	t.SkipNow() // Comment this line out to run the method and create P2PKeys

	for i := 0; i < 1; i++ {
		password := "aaa"

		addr, err := CreateP2PKey("keys", "", password, stchaintypes.SdsNodeP2PAddressPrefix)
		if err != nil {
			t.Fatal(err.Error())
		}

		bechAddr, err := addr.P2pAddressToBech()
		if err != nil {
			t.Fatal(err.Error())
		}

		keyjson, err := ioutil.ReadFile("keys/" + bechAddr + ".json")
		if err != nil {
			t.Fatal(err.Error())
		}
		key, err := DecryptKey(keyjson, password)
		if err != nil {
			t.Fatal(err.Error())
		}

		pubKey := ed25519.PrivKeyBytesToPubKey(key.PrivateKey)
		bechPub, err := bech32.ConvertAndEncode(stchaintypes.SdsNodeP2PPubkeyPrefix, pubKey.Bytes())
		if err != nil {
			t.Fatal(err.Error())
		}
		fmt.Printf("Address: %v  PublicKey: %v\n", bechAddr, bechPub)
	}
}

func TestDecryptP2PKeyJson(t *testing.T) {
	t.SkipNow() // Comment this line out to run the method and decrypt a P2PKey JSON file
	key, err := DecryptKey([]byte("put the content of the P2PKey JSON file here"), "aaa")
	if err != nil {
		t.Fatal(err.Error())
	}

	pubKey := ed25519.PrivKeyBytesToPubKey(key.PrivateKey)
	bechPub, err := bech32.ConvertAndEncode(stchaintypes.SdsNodeP2PPubkeyPrefix, pubKey.Bytes())
	if err != nil {
		t.Fatal(err.Error())
	}

	bechAddr, err := types.BytesToAddress(pubKey.Address().Bytes()).P2pAddressToBech()
	if err != nil {
		t.Fatal(err.Error())
	}

	fmt.Printf("Address: %v  PublicKey: %v\n", bechAddr, bechPub)
}

func TestDecryptWalletJson(t *testing.T) {
	t.SkipNow() // Comment this line out to run the method and decrypt a wallet JSON file
	key, err := DecryptKey([]byte("put the content of the wallet JSON file here"), "aaa")
	if err != nil {
		t.Fatal(err.Error())
	}

	sdkPrivKey := utilsecp256k1.PrivKeyToSdkPrivKey(key.PrivateKey)
	sdkPubKey := sdkPrivKey.PubKey()

	bechPub, err := bech32.ConvertAndEncode(stchaintypes.AccountPubKeyPrefix, sdkPubKey.Bytes())
	if err != nil {
		t.Fatal(err.Error())
	}

	bechAddr, err := types.BytesToAddress(sdkPubKey.Address().Bytes()).WalletAddressToBech()
	if err != nil {
		t.Fatal(err.Error())
	}

	fmt.Printf("Address: %v  PublicKey: %v  HdPath: %v  Mnemonic: %v\n", bechAddr, bechPub, key.HdPath, key.Mnemonic)
}

func TestDecryptPrivValidatorKeyJson(t *testing.T) {
	t.SkipNow() // Comment this line out to run the method and decrypt a priv_validator_key JSON file (SP node validator key)
	p2pKey := privval.FilePVKey{}
	err := tmjson.Unmarshal([]byte("put the content of the priv_validator_key.json file here"), &p2pKey)
	if err != nil {
		t.Fatal(err)
	}

	p2pKeyTm, success := p2pKey.PrivKey.(tmed25519.PrivKey)
	if !success {
		t.Fatal("couldn't convert validator private key to tendermint ed25519")
	}
	pubKey := p2pKeyTm.PubKey()

	bechPub, err := bech32.ConvertAndEncode(stchaintypes.SdsNodeP2PPubkeyPrefix, pubKey.Bytes())
	if err != nil {
		t.Fatal(err.Error())
	}

	sdkPubKey := ed25519.PubKeyBytesToSdkPubKey(pubKey.Bytes())
	apk, err := codectypes.NewAnyWithValue(sdkPubKey)
	if err != nil {
		t.Fatal(err.Error())
	}
	pubKeyJson, err := codec.ProtoMarshalJSON(apk, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	bechAddr, err := types.BytesToAddress(pubKey.Address().Bytes()).P2pAddressToBech()
	if err != nil {
		t.Fatal(err.Error())
	}

	fmt.Printf("Address: %v  PublicKey: %v  PublicKeyJson: %v\n", bechAddr, bechPub, string(pubKeyJson))
}
