package utils

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	utilsecp256k1 "github.com/stratosnet/sds/utils/crypto/secp256k1"
	stchaintypes "github.com/stratosnet/stratos-chain/types"
	tmed25519 "github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/privval"
)

var cdc = codec.NewLegacyAmino()

func init() {
	cryptocodec.RegisterCrypto(cdc)
}

func TestCreateWallet(t *testing.T) {
	t.SkipNow() // Comment this line out to run the method and create a wallet
	password := "aaa"

	mnemonic, err := NewMnemonic()
	if err != nil {
		t.Fatal(err.Error())
	}
	addr, err := CreateWallet("keys", "", password, stchaintypes.StratosBech32Prefix, mnemonic, "", "m/44'/606'/0'/0/44")
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

	sdkPubkey, err := stchaintypes.SdsPubKeyFromByteArr(utilsecp256k1.PrivKeyToPubKey(key.PrivateKey))
	if err != nil {
		t.Fatal(err.Error())
	}
	bechPub, err := stchaintypes.SdsPubKeyToBech32(sdkPubkey)
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
	bechPub := ed25519.PubKeyBytesToSdkPubKey(pubKey.Bytes())
	fmt.Printf("Address: %v  PublicKey: %v", bechAddr, bechPub)
}

func TestDecryptP2PKeyJson(t *testing.T) {
	//t.SkipNow() // Comment this line out to run the method and decrypt a P2PKey JSON file
	key, err := DecryptKey([]byte("{\"address\":\"3eaa7fc3c63d0b0c94943ca190d46f2c06de34af\",\"name\":\"p2pkey\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"ciphertext\":\"050292b6f13dbb00eea81792a4326b6e0754dcc2db5161783d065d89bb505d0a89834e4b20fe1eea8ae06d0b311017462f59b1fb0bad13c40d9dbf8921206e70dd6fcb06d909ea4fa95e6efc31a2340c5fe17c330ff97291218780c67bc438f8ff213d04430fd71a7751e260\",\"cipherparams\":{\"iv\":\"7ef963e012d35723fb9d106829f3d616\"},\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":4096,\"p\":6,\"r\":8,\"salt\":\"d6e54b4a9b90b29a7bae81044d362e9fa7b1cf61175d3a89e73a345b59fc91e2\"},\"mac\":\"40d40cfc8f800c4994af529c92ed7c716fca511eeaa9411d91172babbfe8e8cb\"},\"id\":\"873d1018-d47d-42c0-b632-af889a910427\",\"version\":3}"), "aaa")
	if err != nil {
		t.Fatal(err.Error())
	}

	pubKey := ed25519.PrivKeyBytesToPubKey(key.PrivateKey)
	bechPub := ed25519.PubKeyBytesToSdkPubKey(pubKey.Bytes())

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

	sdkPrivKey := utilsecp256k1.PrivKeyBytesToSdkPriv(key.PrivateKey)
	sdkPubKey := sdkPrivKey.PubKey()

	bechPub := utilsecp256k1.PubKeyBytesToSdkPubKey(sdkPubKey.Bytes())

	bechAddr, err := bech32.ConvertAndEncode(stchaintypes.StratosBech32Prefix, sdkPubKey.Address().Bytes())
	if err != nil {
		t.Fatal(err.Error())
	}

	fmt.Printf("Address: %v  PublicKey: %v  HdPath: %v  Mnemonic: %v", bechAddr, bechPub, key.HdPath, key.Mnemonic)
}

func TestDecryptPrivValidatorKeyJson(t *testing.T) {
	t.SkipNow() // Comment this line out to run the method and decrypt a priv_validator_key JSON file (SP node validator key)
	p2pKey := privval.FilePVKey{}
	err := cdc.UnmarshalJSON([]byte("put the content of the priv_validator_key.json file here"), &p2pKey)
	if err != nil {
		t.Fatal(err)
	}

	p2pKeyTm, success := p2pKey.PrivKey.(tmed25519.PrivKey)
	if !success {
		t.Fatal("couldn't convert validator private key to tendermint ed25519")
	}
	pubKey := p2pKeyTm.PubKey()

	bechPub := ed25519.PubKeyBytesToSdkPubKey(pubKey.Bytes())

	bechAddr, err := bech32.ConvertAndEncode(stchaintypes.SdsNodeP2PAddressPrefix, pubKey.Address().Bytes())
	if err != nil {
		t.Fatal(err.Error())
	}

	fmt.Printf("Address: %v  PublicKey: %v", bechAddr, bechPub)
}
