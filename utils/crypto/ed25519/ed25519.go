package ed25519

import (
	"github.com/stratosnet/sds/utils/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

func NewKey() []byte {
	privKey := ed25519.GenPrivKey()
	return privKey[:]
}

func PrivKeyBytesToPrivKey(privKey []byte) crypto.PrivKey {
	var privKey2 [64]byte
	copy(privKey2[:], privKey)
	return ed25519.PrivKeyEd25519(privKey2)
}

func PrivKeyBytesToPubKey(privKey []byte) crypto.PubKey {
	pubKey := PrivKeyBytesToPrivKey(privKey).PubKey()
	pubKey2 := pubKey.(ed25519.PubKeyEd25519)
	return pubKey2
}

func PrivKeyBytesToPubKeyBytes(privKey []byte) []byte {
	pubKey := PrivKeyBytesToPrivKey(privKey).PubKey()
	pubKey2 := pubKey.(ed25519.PubKeyEd25519)
	return pubKey2[:]
}

func PrivKeyBytesToAddress(privKey []byte) types.Address {
	address := PrivKeyBytesToPrivKey(privKey).PubKey().Address()
	return types.BytesToAddress(address)
}

func PubKeyBytesToPubKey(pubKey []byte) crypto.PubKey {
	var pubKey2 [ed25519.PubKeyEd25519Size]byte
	copy(pubKey2[:], pubKey)
	return ed25519.PubKeyEd25519(pubKey2)
}

func PubKeyBytesToAddress(pubKey []byte) types.Address {
	address := PubKeyBytesToPubKey(pubKey).Address()
	return types.BytesToAddress(address)
}
