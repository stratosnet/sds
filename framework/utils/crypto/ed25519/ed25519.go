package ed25519

import (
	"github.com/stratosnet/sds/framework/crypto/ed25519"
	fwcryptotypes "github.com/stratosnet/sds/framework/crypto/types"
	"github.com/stratosnet/sds/framework/utils/types"
)

func NewKey() []byte {
	privKey := ed25519.GenPrivKey()
	return privKey.Bytes()
}

func PrivKeyBytesToPrivKey(privKey []byte) fwcryptotypes.PrivKey {
	return ed25519.Generate(privKey)
}

func PrivKeyBytesToPubKey(privKey []byte) fwcryptotypes.PubKey {
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

func PubKeyBytesToPubKey(pubKey []byte) fwcryptotypes.PubKey {
	var pubKey2 []byte
	copy(pubKey2[:], pubKey)
	return &ed25519.PubKey{Key: pubKey2}
}

func PubKeyBytesToAddress(pubKey []byte) types.Address {
	address := PubKeyBytesToPubKey(pubKey).Address()
	return types.BytesToAddress(address)
}

func PrivKeyBytesToSdkPrivKey(privKey []byte) fwcryptotypes.PrivKey {
	return ed25519.Generate(privKey)
}

func PrivKeyBytesToSdkPubKey(privKey []byte) fwcryptotypes.PubKey {
	pubKey := PrivKeyBytesToSdkPrivKey(privKey).PubKey()
	return pubKey
}

func PubKeyBytesToSdkPubKey(pubKey []byte) fwcryptotypes.PubKey {
	retPubKey := ed25519.PubKey{Key: pubKey}
	return &retPubKey
}
