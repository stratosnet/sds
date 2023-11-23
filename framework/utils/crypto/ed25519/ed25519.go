package ed25519

import (
	"github.com/stratosnet/framework/utils/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"

	sdked25519 "github.com/stratosnet/tx-client/crypto/ed25519"
	cryptotypes "github.com/stratosnet/tx-client/crypto/types"
)

func NewKey() []byte {
	privKey := ed25519.GenPrivKey()
	return privKey[:]
}

func PrivKeyBytesToPrivKey(privKey []byte) crypto.PrivKey {
	var privKey2 [64]byte
	copy(privKey2[:], privKey)
	return ed25519.PrivKey(privKey2[:])
}

func PrivKeyBytesToPubKey(privKey []byte) crypto.PubKey {
	pubKey := PrivKeyBytesToPrivKey(privKey).PubKey()
	pubKey2 := pubKey.(ed25519.PubKey)
	return pubKey2
}

func PrivKeyBytesToPubKeyBytes(privKey []byte) []byte {
	pubKey := PrivKeyBytesToPrivKey(privKey).PubKey()
	pubKey2 := pubKey.(ed25519.PubKey)
	return pubKey2[:]
}

func PrivKeyBytesToAddress(privKey []byte) types.Address {
	address := PrivKeyBytesToPrivKey(privKey).PubKey().Address()
	return types.BytesToAddress(address)
}

func PubKeyBytesToPubKey(pubKey []byte) crypto.PubKey {
	var pubKey2 [ed25519.PubKeySize]byte
	copy(pubKey2[:], pubKey)
	return ed25519.PubKey(pubKey2[:])
}

func PubKeyBytesToAddress(pubKey []byte) types.Address {
	address := PubKeyBytesToPubKey(pubKey).Address()
	return types.BytesToAddress(address)
}

func PrivKeyBytesToSdkPrivKey(privKey []byte) cryptotypes.PrivKey {
	retPrivKey := sdked25519.PrivKey{Key: privKey} //nolint:staticcheck
	return &retPrivKey
}

func PrivKeyBytesToSdkPubKey(privKey []byte) cryptotypes.PubKey {
	pubKey := PrivKeyBytesToSdkPrivKey(privKey).PubKey()
	return pubKey
}

func PubKeyBytesToSdkPubKey(pubKey []byte) cryptotypes.PubKey {
	retPubKey := sdked25519.PubKey{Key: pubKey}
	return &retPubKey
}
