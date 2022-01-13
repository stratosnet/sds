package utils

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	"github.com/tendermint/go-amino"
	tmed25519 "github.com/tendermint/tendermint/crypto/ed25519"
	cryptoamino "github.com/tendermint/tendermint/crypto/encoding/amino"
	"github.com/tendermint/tendermint/libs/bech32"
	"github.com/tendermint/tendermint/privval"
)

var cdc = amino.NewCodec()

func init() {
	cryptoamino.RegisterAmino(cdc)
}

func TestCreateWallet(t *testing.T) {
	t.SkipNow() // Comment this line out to run the method and create a wallet
	password := "aaa"
	hrp := "st"

	mnemonic, err := NewMnemonic()
	if err != nil {
		t.Fatal(err.Error())
	}
	addr, err := CreateWallet("keys", "", password, hrp, mnemonic, "passphrase", "44'/606'/0'/0/44")
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
	t.SkipNow() // Comment this line out to run the method and create a P2PKey
	password := "aaa"
	hrp := "stsdsp2p"

	addr, err := CreateP2PKey("keys", "", password, hrp)
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
	t.SkipNow() // Comment this line out to run the method and decrypt a P2PKey JSON file
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

func TestDecryptWalletJson(t *testing.T) {
	t.SkipNow() // Comment this line out to run the method and decrypt a wallet JSON file
	hrp := "st"
	key, err := DecryptKey([]byte("put the content of the wallet JSON file here"), "aaa")
	if err != nil {
		t.Fatal(err.Error())
	}

	tmPrivKey := secp256k1.PrivKeyBytesToTendermint(key.PrivateKey)
	tmPubKey := tmPrivKey.PubKey()

	bechPub, err := bech32.ConvertAndEncode(hrp+"pub", tmPubKey.Bytes())
	if err != nil {
		t.Fatal(err.Error())
	}

	bechAddr, err := bech32.ConvertAndEncode(hrp, tmPubKey.Address().Bytes())
	if err != nil {
		t.Fatal(err.Error())
	}

	fmt.Printf("Address: %v  PublicKey: %v  HdPath: %v", bechAddr, bechPub, key.HdPath)
}

func TestDecryptPrivValidatorKeyJson(t *testing.T) {
	t.SkipNow() // Comment this line out to run the method and decrypt a priv_validator_key JSON file (SP node validator key)
	hrp := "stsdsp2p"
	p2pKey := privval.FilePVKey{}
	err := cdc.UnmarshalJSON([]byte("put the content of the priv_validator_key.json file here"), &p2pKey)
	if err != nil {
		t.Fatal(err)
	}

	p2pKeyTm, success := p2pKey.PrivKey.(tmed25519.PrivKeyEd25519)
	if !success {
		t.Fatal("couldn't convert validator private key to tendermint ed25519")
	}
	pubKey := p2pKeyTm.PubKey()

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
