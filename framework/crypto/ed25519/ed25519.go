package ed25519

import (
	"crypto/ed25519"
	"crypto/subtle"
	"io"

	"github.com/hdevalence/ed25519consensus"

	tmcrypto "github.com/stratosnet/sds/framework/crypto/tendermint"
	"github.com/stratosnet/sds/framework/crypto/tendermint/tmhash"
	fwcryptotypes "github.com/stratosnet/sds/framework/crypto/types"
)

const (
	// PubKeySize is is the size, in bytes, of public keys as used in this package.
	PubKeySize = 32
	// PrivKeySize is the size, in bytes, of private keys as used in this package.
	PrivKeySize = 64
	// Size of an Edwards25519 signature. Namely the size of a compressed
	// Edwards25519 point, and a field element. Both of which are 32 bytes.
	SignatureSize = 64
	// SeedSize is the size, in bytes, of private key seeds. These are the
	// private key representations used by RFC 8032.
	SeedSize = 32

	KeyType = "ed25519"
)

var (
	_ fwcryptotypes.PrivKey = &PrivKey{}
)

// Generate generates an ed25519 private key from the given bytes.
func Generate(bz []byte) fwcryptotypes.PrivKey {
	bzArr := make([]byte, PrivKeySize)
	copy(bzArr, bz)

	return &PrivKey{
		Key: bzArr,
	}
}

// Bytes returns the privkey byte format.
func (privKey *PrivKey) Bytes() []byte {
	return privKey.Key
}

// Sign produces a signature on the provided message.
// This assumes the privkey is wellformed in the golang format.
// The first 32 bytes should be random,
// corresponding to the normal ed25519 private key.
// The latter 32 bytes should be the compressed public key.
// If these conditions aren't met, Sign will panic or produce an
// incorrect signature.
func (privKey *PrivKey) Sign(msg []byte) ([]byte, error) {
	return ed25519.Sign(privKey.Key, msg), nil
}

// PubKey gets the corresponding public key from the private key.
//
// Panics if the private key is not initialized.
func (privKey *PrivKey) PubKey() fwcryptotypes.PubKey {
	// If the latter 32 bytes of the privkey are all zero, privkey is not
	// initialized.
	initialized := false
	for _, v := range privKey.Key[32:] {
		if v != 0 {
			initialized = true
			break
		}
	}

	if !initialized {
		panic("Expected ed25519 PrivKey to include concatenated pubkey bytes")
	}

	pubkeyBytes := make([]byte, PubKeySize)
	copy(pubkeyBytes, privKey.Key[32:])
	return &PubKey{Key: pubkeyBytes}
}

// Equals - you probably don't need to use this.
// Runs in constant time based on length of the keys.
func (privKey *PrivKey) Equals(other fwcryptotypes.LedgerPrivKey) bool {
	if privKey.Type() != other.Type() {
		return false
	}

	return subtle.ConstantTimeCompare(privKey.Bytes(), other.Bytes()) == 1
}

func (privKey *PrivKey) Type() string {
	return KeyType
}

// GenPrivKey generates a new ed25519 private key. These ed25519 keys must not
// be used in SDK apps except in a tendermint validator context.
// It uses OS randomness in conjunction with the current global random seed
// in tendermint/libs/common to generate the private key.
func GenPrivKey() *PrivKey {
	return genPrivKey(tmcrypto.CReader())
}

// genPrivKey generates a new ed25519 private key using the provided reader.
func genPrivKey(rand io.Reader) *PrivKey {
	seed := make([]byte, SeedSize)

	_, err := io.ReadFull(rand, seed)
	if err != nil {
		panic(err)
	}

	return &PrivKey{Key: ed25519.NewKeyFromSeed(seed)}
}

// GenPrivKeyFromSecret hashes the secret with SHA2, and uses
// that 32 byte output to create the private key.
// NOTE: ed25519 keys must not be used in SDK apps except in a tendermint validator context.
// NOTE: secret should be the output of a KDF like bcrypt,
// if it's derived from user input.
func GenPrivKeyFromSecret(secret []byte) *PrivKey {
	seed := tmcrypto.Sha256(secret) // Not Ripemd160 because we want 32 bytes.

	return &PrivKey{Key: ed25519.NewKeyFromSeed(seed)}
}

//-------------------------------------

var (
	_ fwcryptotypes.PubKey = &PubKey{}
)

// Address is the SHA256-20 of the raw pubkey bytes.
// It doesn't implement ADR-28 addresses and it must not be used
// in SDK except in a tendermint validator context.
func (pubKey *PubKey) Address() fwcryptotypes.Address {
	if len(pubKey.Key) != PubKeySize {
		panic("pubkey is incorrect size")
	}
	// For ADR-28 compatible address we would need to
	// return address.Hash(proto.MessageName(pubKey), pubKey.Key)
	return fwcryptotypes.Address(tmhash.SumTruncated(pubKey.Key))
}

// Bytes returns the PubKey byte format.
func (pubKey *PubKey) Bytes() []byte {
	return pubKey.Key
}

func (pubKey *PubKey) VerifySignature(msg []byte, sig []byte) bool {
	// make sure we use the same algorithm to sign
	if len(sig) != SignatureSize {
		return false
	}

	// uses https://github.com/hdevalence/ed25519consensus.Verify to comply with zip215 verification rules
	return ed25519consensus.Verify(pubKey.Key, msg, sig)
}

//// String returns Hex representation of a pubkey with it's type
//func (pubKey *PubKey) String() string {
//	return fmt.Sprintf("PubKeyEd25519{%X}", pubKey.Key)
//}

func (pubKey *PubKey) Type() string {
	return KeyType
}

func (pubKey *PubKey) Equals(other fwcryptotypes.PubKey) bool {
	if pubKey.Type() != other.Type() {
		return false
	}

	return subtle.ConstantTimeCompare(pubKey.Bytes(), other.Bytes()) == 1
}
