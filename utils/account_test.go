package utils

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	"io/ioutil"
	"testing"

	"github.com/stratosnet/sds/utils/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/bech32"
)

func TestCreateWallet(t *testing.T) {
	password := "aaa"
	hrp := "st"

	mnemonic, err := NewMnemonic()
	if err != nil {
		t.Fatal(err.Error())
	}
	addr, err := CreateWallet("keys", "", password, hrp, mnemonic, "passphrase", "44'/606'/0'/0/44", 4096, 6)
	if err != nil {
		t.Fatal(err.Error())
	}

	bechAddr, err := addr.ToBech(hrp)
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

	privKey := secp256k1.PrivKeyBytesToTendermint(key.PrivateKey)
	pubKey := privKey.PubKey()
	bechPub, err := types.Bech32ifyPubKey(types.Bech32PubKeyTypeAccPub, pubKey)
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Printf("Address: %v  PublicKey: %v  Mnemonic: %v\n", bechAddr, bechPub, key.Mnemonic)
}

func TestCreateP2PKey(t *testing.T) {
	password := "aaa"
	hrp := "stsdsp2p"

	addr, err := CreateP2PKey("keys", "", password, hrp, 4096, 6)
	if err != nil {
		t.Fatal(err.Error())
	}

	bechAddr, err := addr.ToBech(hrp)
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
	bechPub, err := bech32.ConvertAndEncode(hrp, pubKey.Bytes())
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Printf("Address: %v  PublicKey: %v", bechAddr, bechPub)
}

func TestDecryptP2PKeyJson(t *testing.T) {
	hrp := "stsdsp2p"
	key, err := DecryptKey([]byte("put the content of the P2P key JSON file here"), "aaa")
	if err != nil {
		t.Fatal(err.Error())
	}

	pubKey := ed25519.PrivKeyBytesToPubKey(key.PrivateKey)
	bechPub, err := bech32.ConvertAndEncode(hrp, pubKey.Bytes())
	if err != nil {
		t.Fatal(err.Error())
	}

	bechAddr, err := bech32.ConvertAndEncode(hrp, pubKey.Address().Bytes())
	if err != nil {
		t.Fatal(err.Error())
	}

	fmt.Printf("Address: %v  PublicKey: %v", bechAddr, bechPub)
}
