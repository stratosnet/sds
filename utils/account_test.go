package utils

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stratosnet/sds/utils/crypto/ed25519"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	"github.com/stratosnet/sds/utils/types"
	stchaintypes "github.com/stratosnet/stratos-chain/types"
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

	mnemonic, err := NewMnemonic()
	if err != nil {
		t.Fatal(err.Error())
	}
	addr, err := CreateWallet("keys", "", password, stchaintypes.StratosBech32Prefix, mnemonic, "passphrase", "44'/606'/0'/0/44")
	if err != nil {
		t.Fatal(err.Error())
	}

	bechAddr, err := addr.ToBech(stchaintypes.StratosBech32Prefix)
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
	bechPub, err := stchaintypes.Bech32ifyPubKey(stchaintypes.Bech32PubKeyTypeAccPub, pubKey)
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Printf("Address: %v  PublicKey: %v  Mnemonic: %v\n", bechAddr, bechPub, key.Mnemonic)
}

func TestCreateP2PKey(t *testing.T) {
	t.SkipNow() // Comment this line out to run the method and create a P2PKey
	password := "aaa"

	addr, err := CreateP2PKey("keys", "", password, stchaintypes.SdsNodeP2PAddressPrefix)
	if err != nil {
		t.Fatal(err.Error())
	}

	bechAddr, err := types.P2pAddressToBech(addr)
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
	bechPub, err := stchaintypes.Bech32ifyPubKey(stchaintypes.Bech32PubKeyTypeAccPub, pubKey)
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Printf("Address: %v  PublicKey: %v", bechAddr, bechPub)
}

func TestDecryptP2PKeyJson(t *testing.T) {
	t.SkipNow() // Comment this line out to run the method and decrypt a P2PKey JSON file
	key, err := DecryptKey([]byte("put the content of the P2P key JSON file here"), "aaa")
	if err != nil {
		t.Fatal(err.Error())
	}

	pubKey := ed25519.PrivKeyBytesToPubKey(key.PrivateKey)
	bechPub, err := stchaintypes.Bech32ifyPubKey(stchaintypes.Bech32PubKeyTypeSdsP2PPub, pubKey)
	if err != nil {
		t.Fatal(err.Error())
	}

	bechAddr, err := bech32.ConvertAndEncode(stchaintypes.SdsNodeP2PAddressPrefix, pubKey.Address().Bytes())
	if err != nil {
		t.Fatal(err.Error())
	}

	fmt.Printf("Address: %v  PublicKey: %v", bechAddr, bechPub)
}

func TestDecryptWalletJson(t *testing.T) {
	t.SkipNow() // Comment this line out to run the method and decrypt a wallet JSON file
	key, err := DecryptKey([]byte("put the content of the wallet JSON file here"), "aaa")
	if err != nil {
		t.Fatal(err.Error())
	}

	tmPrivKey := secp256k1.PrivKeyBytesToTendermint(key.PrivateKey)
	tmPubKey := tmPrivKey.PubKey()

	bechPub, err := stchaintypes.Bech32ifyPubKey(stchaintypes.Bech32PubKeyTypeAccPub, tmPubKey)
	if err != nil {
		t.Fatal(err.Error())
	}

	bechAddr, err := bech32.ConvertAndEncode(stchaintypes.StratosBech32Prefix, tmPubKey.Address().Bytes())
	if err != nil {
		t.Fatal(err.Error())
	}

	fmt.Printf("Address: %v  PublicKey: %v  HdPath: %v", bechAddr, bechPub, key.HdPath)
}

func TestDecryptPrivValidatorKeyJson(t *testing.T) {
	t.SkipNow() // Comment this line out to run the method and decrypt a priv_validator_key JSON file (SP node validator key)
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

	bechPub, err := stchaintypes.Bech32ifyPubKey(stchaintypes.Bech32PubKeyTypeSdsP2PPub, pubKey)
	if err != nil {
		t.Fatal(err.Error())
	}

	bechAddr, err := bech32.ConvertAndEncode(stchaintypes.SdsNodeP2PAddressPrefix, pubKey.Address().Bytes())
	if err != nil {
		t.Fatal(err.Error())
	}

	fmt.Printf("Address: %v  PublicKey: %v", bechAddr, bechPub)
}
