package secp256k1

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/subtle"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/tyler-smith/go-bip39"

	ethcrypto "github.com/stratosnet/framework/crypto/ethereum"
	"github.com/stratosnet/framework/crypto/ethereum/common"
	cryptotypes "github.com/stratosnet/framework/crypto/types"
)

//-----------------------------------------------------------------------------------------------

const (
	// PrivKeySize defines the size of the PrivKey bytes
	PrivKeySize = 32
	// PubKeySize defines the size of the PubKey bytes
	PubKeySize = 33
	// KeyType is the string constant for the Secp256k1 algorithm
	KeyType = "eth_secp256k1"
)

// ----------------------------------------------------------------------------
// secp256k1 Private Key

var (
	_ cryptotypes.PrivKey = &PrivKey{}
)

// Generate generates an eth_secp256k1 private key from the given bytes.
func Generate(bz []byte) cryptotypes.PrivKey {
	bzArr := make([]byte, PrivKeySize)
	copy(bzArr, bz)

	return &PrivKey{
		Key: bzArr,
	}
}

// Derive derives and returns the eth_secp256k1 private key for the given mnemonic and HD path.
func Derive(mnemonic, bip39Passphrase, path string) ([]byte, error) {
	hdpath, err := common.ParseDerivationPath(path)
	if err != nil {
		return nil, err
	}

	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, bip39Passphrase)
	if err != nil {
		return nil, err
	}

	// create a BTC-utils hd-derivation key chain
	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}

	key := masterKey
	for _, n := range hdpath {
		key, err = key.Derive(n)
		if err != nil {
			return nil, err
		}
	}

	// btc-utils representation of a secp256k1 private key
	privateKey, err := key.ECPrivKey()
	if err != nil {
		return nil, err
	}

	// cast private key to a convertible form (single scalar field element of secp256k1)
	// and then load into ethcrypto private key format.
	// TODO: add links to godocs of the two methods or implementations of them, to compare equivalency
	privateKeyECDSA := privateKey.ToECDSA()
	derivedKey := ethcrypto.FromECDSA(privateKeyECDSA)

	return derivedKey, nil
}

// GenerateKey generates a new random private key. It returns an error upon
// failure.
func GenerateKey() (*PrivKey, error) {
	priv, err := ethcrypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	return &PrivKey{
		Key: ethcrypto.FromECDSA(priv),
	}, nil
}

func MakePubKey(key []byte) cryptotypes.PubKey {
	return &PubKey{Key: key}
}

// Bytes returns the byte representation of the ECDSA Private Key.
func (privKey *PrivKey) Bytes() []byte {
	bz := make([]byte, len(privKey.Key))
	copy(bz, privKey.Key)

	return bz
}

// PubKey returns the ECDSA private key's public key. If the privkey is not valid
// it returns a nil value.
func (privKey *PrivKey) PubKey() cryptotypes.PubKey {
	ecdsaPrivKey, err := privKey.ToECDSA()
	if err != nil {
		return nil
	}

	return &PubKey{
		Key: ethcrypto.CompressPubkey(&ecdsaPrivKey.PublicKey),
	}
}

// Equals returns true if two ECDSA private keys are equal and false otherwise.
func (privKey *PrivKey) Equals(other cryptotypes.LedgerPrivKey) bool {
	return privKey.Type() == other.Type() && subtle.ConstantTimeCompare(privKey.Bytes(), other.Bytes()) == 1
}

// Type returns eth_secp256k1
func (privKey *PrivKey) Type() string {
	return KeyType
}

// Sign creates a recoverable ECDSA signature on the secp256k1 curve over the
// provided hash of the message. The produced signature is 65 bytes
// where the last byte contains the recovery ID.
func (privKey *PrivKey) Sign(digestBz []byte) ([]byte, error) {
	// TODO: remove
	if len(digestBz) != ethcrypto.DigestLength {
		digestBz = ethcrypto.Keccak256Hash(digestBz).Bytes()
	}

	key, err := privKey.ToECDSA()
	if err != nil {
		return nil, err
	}

	return ethcrypto.Sign(digestBz, key)
}

// ToECDSA returns the ECDSA private key as a reference to ecdsa.PrivateKey type.
func (privKey *PrivKey) ToECDSA() (*ecdsa.PrivateKey, error) {
	return ethcrypto.ToECDSA(privKey.Bytes())
}

// ----------------------------------------------------------------------------
// secp256k1 Public Key

var (
	_ cryptotypes.PubKey = &PubKey{}
)

// Address returns the address of the ECDSA public key.
// The function will return an empty address if the public key is invalid.
func (pubKey *PubKey) Address() cryptotypes.Address {
	pubk, err := ethcrypto.DecompressPubkey(pubKey.Key)
	if err != nil {
		return nil
	}

	return cryptotypes.Address(ethcrypto.PubkeyToAddress(*pubk).Bytes())
}

// Bytes returns the raw bytes of the ECDSA public key.
func (pubKey *PubKey) Bytes() []byte {
	bz := make([]byte, len(pubKey.Key))
	copy(bz, pubKey.Key)

	return bz
}

//// String implements the fmt.Stringer interface.
//func (pubKey *PubKey) String() string {
//	return fmt.Sprintf("EthPubKeySecp256k1{%X}", pubKey.Key)
//}

// Type returns eth_secp256k1
func (pubKey *PubKey) Type() string {
	return KeyType
}

// Equals returns true if the pubkey type is the same and their bytes are deeply equal.
func (pubKey *PubKey) Equals(other cryptotypes.PubKey) bool {
	return pubKey.Type() == other.Type() && bytes.Equal(pubKey.Bytes(), other.Bytes())
}

// VerifySignature verifies that the ECDSA public key created a given signature over
// the provided message. It will calculate the Keccak256 hash of the message
// prior to verification.
//
// CONTRACT: The signature should be in [R || S] format.
func (pubKey *PubKey) VerifySignature(msg, sig []byte) bool {
	if len(sig) == ethcrypto.SignatureLength {
		// remove recovery ID (V) if contained in the signature
		sig = sig[:len(sig)-1]
	}

	// the signature needs to be in [R || S] format when provided to VerifySignature
	return ethcrypto.VerifySignature(pubKey.Key, ethcrypto.Keccak256Hash(msg).Bytes(), sig)
}
