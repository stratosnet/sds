package secp256k1

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"

	"github.com/btcsuite/btcd/btcec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	ethsecp256k1 "github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/stratosnet/sds/utils/types"
	chainethsecp256k1 "github.com/stratosnet/stratos-chain/crypto/ethsecp256k1"
	"github.com/stratosnet/stratos-chain/crypto/hd"
	tmsecp256k1 "github.com/tendermint/tendermint/crypto/secp256k1"
)

func PubKeyToTendermint(pubKey ecdsa.PublicKey) (tmsecp256k1.PubKey, error) {
	compressed := ethsecp256k1.CompressPubkey(pubKey.X, pubKey.Y)
	return PubKeyBytesToTendermint(compressed)
}

func PubKeyBytesToTendermint(pubKey []byte) (tmsecp256k1.PubKey, error) {
	if !btcec.IsCompressedPubKey(pubKey) {
		pubKeyObject, err := UnmarshalPubkey(pubKey)
		if err != nil {
			fixedSizedBytes := [33]byte{}
			return fixedSizedBytes[:], err
		}
		pubKey = ethsecp256k1.CompressPubkey(pubKeyObject.X, pubKeyObject.Y)
	}
	var compressedArr [33]byte
	copy(compressedArr[:], pubKey)
	return compressedArr[:], nil
}

func PrivKeyBytesToTendermint(privKey []byte) tmsecp256k1.PrivKey {
	var bzArr [32]byte
	copy(bzArr[:], privKey)
	return bzArr[:]
}

// PrivKeyToPubKey returns the public key associated with the given private key in the uncompressed format.
func PrivKeyToPubKey(privKey []byte) []byte {
	_, pubKeyObject := btcec.PrivKeyFromBytes(ethsecp256k1.S256(), privKey[:])
	return pubKeyObject.SerializeUncompressed()
}

// PrivKeyToPubKeyCompressed returns the public key associated with the given private key in the compressed format.
func PrivKeyToPubKeyCompressed(privKey []byte) []byte {
	_, pubKeyObject := btcec.PrivKeyFromBytes(ethsecp256k1.S256(), privKey[:])
	return pubKeyObject.SerializeCompressed()
}

// PrivKeyToAddress calculates the wallet address from the user's private key
func PrivKeyToAddress(privKey []byte) types.Address {
	ethPrivKey := hd.EthSecp256k1.Generate()(privKey)
	return types.BytesToAddress(ethPrivKey.PubKey().Address())
}

// UnmarshalPubkey converts bytes to a secp256k1 public key.
func UnmarshalPubkey(pub []byte) (*ecdsa.PublicKey, error) {
	x, y := elliptic.Unmarshal(ethsecp256k1.S256(), pub)
	if x == nil {
		return nil, errors.New("invalid secp256k1 public key")
	}
	return &ecdsa.PublicKey{Curve: ethsecp256k1.S256(), X: x, Y: y}, nil
}

func PrivKeyBytesToSdkPriv(privKey []byte) cryptotypes.PrivKey {
	return &chainethsecp256k1.PrivKey{Key: privKey}
}

func PubKeyBytesToSdkPubKey(pubKey []byte) cryptotypes.PubKey {
	retPubKey := chainethsecp256k1.PubKey{Key: pubKey}
	return &retPubKey
}
