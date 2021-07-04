package ed25519

import (
	"crypto/ed25519"
	"github.com/stratosnet/sds/utils/crypto"
	"github.com/stratosnet/sds/utils/types"
)

func PrivateKeyToPublicKey(privateKey []byte) []byte {
	privateKey2 := ed25519.PrivateKey(privateKey)
	publicKey := privateKey2.Public().(ed25519.PublicKey)
	return publicKey
}

func PrivateKeyToAddress(privateKey []byte) types.Address {
	publicKey := PrivateKeyToPublicKey(privateKey)
	return types.BytesToAddress(crypto.Keccak256(publicKey)[12:])
}
