package types

import (
	"github.com/stratosnet/sds/framework/crypto/ed25519"
	"github.com/stratosnet/sds/framework/crypto/secp256k1"
	fwcryptotypes "github.com/stratosnet/sds/framework/crypto/types"
	"github.com/stratosnet/sds/framework/types/bech32"
)

// WalletPubKeyFromBech32 returns an secp256k1 AccPublicKey from a Bech32 string.
func WalletPubKeyFromBech32(pubkeyStr string) (fwcryptotypes.PubKey, error) {
	_, sdsPubKeyBytes, err := bech32.DecodeAndConvert(pubkeyStr)
	if err != nil {
		return nil, err
	}
	pubKey := secp256k1.PubKey{Key: sdsPubKeyBytes}
	return &pubKey, nil
}

// WalletPubKeyToBech32 convert a AccPublicKey to a Bech32 string.
func WalletPubKeyToBech32(pubkey fwcryptotypes.PubKey) (string, error) {
	return bech32.ConvertAndEncode(WalletPubKeyPrefix, pubkey.Bytes())
}

// P2PPubKeyFromBech32 returns an ed25519 SdsPublicKey from a Bech32 string.
func P2PPubKeyFromBech32(pubkeyStr string) (fwcryptotypes.PubKey, error) {
	_, sdsPubKeyBytes, err := bech32.DecodeAndConvert(pubkeyStr)
	if err != nil {
		return nil, err
	}
	pubKey := ed25519.PubKey{Key: sdsPubKeyBytes}
	return &pubKey, nil
}

// P2PPubKeyToBech32 convert a SdsPublicKey to a Bech32 string.
func P2PPubKeyToBech32(pubkey fwcryptotypes.PubKey) (string, error) {
	return bech32.ConvertAndEncode(P2PPubkeyPrefix, pubkey.Bytes())
}
