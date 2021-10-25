package encryption

import (
	"errors"
	"fmt"
	"github.com/oasisprotocol/ed25519/extra/x25519"
	"golang.org/x/crypto/curve25519"
)

// ECDH transform all Ed25519 points to Curve25519 points and performs a Diffie-Hellman handshake
// to derive a shared key. It throws an error should the Ed25519 points be invalid.
func ECDH(ourPrivateKey, peerPublicKey []byte) ([]byte, error) {
	privateX25519 := x25519.EdPrivateKeyToX25519(ourPrivateKey)
	publicX25519, success := x25519.EdPublicKeyToX25519(peerPublicKey)
	if !success {
		return nil, errors.New("got an invalid ed25519 public key")
	}

	shared, err := curve25519.X25519(privateX25519, publicX25519)
	if err != nil {
		return nil, fmt.Errorf("could not derive a shared key: %w", err)
	}

	return shared, nil
}
