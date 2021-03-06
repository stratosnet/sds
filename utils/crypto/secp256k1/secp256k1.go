package secp256k1

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/utils/types"
)

func PrivKeyToSdkPrivKey(privKey []byte) cryptotypes.PrivKey {
	return hd.Secp256k1.Generate()(privKey)
}

// PrivKeyToPubKey returns the public key associated with the given private key
func PrivKeyToPubKey(privKey []byte) cryptotypes.PubKey {
	return PrivKeyToSdkPrivKey(privKey).PubKey()
}

// PrivKeyToAddress calculates the wallet address from the user's private key
func PrivKeyToAddress(privKey []byte) types.Address {
	privKeyObject := PrivKeyToSdkPrivKey(privKey)
	return types.BytesToAddress(privKeyObject.PubKey().Address())
}

// PubKeyToSdkPubKey converts pubKey bytes to a secp256k1 public key.
func PubKeyToSdkPubKey(pubKey []byte) (cryptotypes.PubKey, error) {
	ecdsaPubKey, err := btcec.ParsePubKey(pubKey, btcec.S256()) // Works for both compressed and uncompressed pubkey formats
	if err != nil {
		return nil, errors.Wrap(err, "invalid secp256k1 public key")
	}
	return &secp256k1.PubKey{Key: ecdsaPubKey.SerializeCompressed()}, nil
}

func PubKeyToAddress(pubKey []byte) (*types.Address, error) {
	pubKeyObject, err := PubKeyToSdkPubKey(pubKey)
	if err != nil {
		return nil, err
	}
	address := types.BytesToAddress(pubKeyObject.Address())
	return &address, nil
}
