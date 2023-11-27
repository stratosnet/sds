package ed25519

import (
	"github.com/stratosnet/tx-client/crypto/ed25519"
	cryptotypes "github.com/stratosnet/tx-client/crypto/types"

	"github.com/stratosnet/framework/utils/types"
)

func NewKey() []byte {
	privKey := ed25519.GenPrivKey()
	return privKey.Bytes()
}

func PrivKeyBytesToPrivKey(privKey []byte) cryptotypes.PrivKey {
	return ed25519.Generate(privKey)
}

func PrivKeyBytesToPubKey(privKey []byte) cryptotypes.PubKey {
	pubKey := PrivKeyBytesToPrivKey(privKey).PubKey()
	return pubKey
}

func PrivKeyBytesToPubKeyBytes(privKey []byte) []byte {
	pubKey := PrivKeyBytesToPrivKey(privKey).PubKey()
	return pubKey.Bytes()
}

func PrivKeyBytesToAddress(privKey []byte) types.Address {
	address := PrivKeyBytesToPrivKey(privKey).PubKey().Address()
	return types.BytesToAddress(address)
}

func PubKeyBytesToPubKey(pubKey []byte) cryptotypes.PubKey {
	var pubKey2 []byte
	copy(pubKey2[:], pubKey)
	return &ed25519.PubKey{Key: pubKey2}
}

func PubKeyBytesToAddress(pubKey []byte) types.Address {
	address := PubKeyBytesToPubKey(pubKey).Address()
	return types.BytesToAddress(address)
}

func PrivKeyBytesToSdkPrivKey(privKey []byte) cryptotypes.PrivKey {
	return ed25519.Generate(privKey)
}

func PrivKeyBytesToSdkPubKey(privKey []byte) cryptotypes.PubKey {
	pubKey := PrivKeyBytesToSdkPrivKey(privKey).PubKey()
	return pubKey
}

func PubKeyBytesToSdkPubKey(pubKey []byte) cryptotypes.PubKey {
	retPubKey := ed25519.PubKey{Key: pubKey}
	return &retPubKey
}
