package types

import (
	"encoding/hex"

	"github.com/stratosnet/sds/framework/crypto/ed25519"
	"github.com/stratosnet/sds/framework/crypto/secp256k1"
	"github.com/stratosnet/sds/framework/utils"
)

// VerifyWalletSignBytes []byte version of VerifyWalletKey() for pubkey and signature format
func VerifyWalletSignBytes(walletPubkey []byte, signature []byte, message string) bool {
	pubkey := secp256k1.PubKey{Key: walletPubkey}
	return pubkey.VerifySignature([]byte(message), signature)
}

// VerifyP2pSignBytes verify the signature made by P2P key
func VerifyP2pSignBytes(p2pPubkey []byte, signature []byte, message []byte) bool {
	pk := ed25519.PubKey{Key: p2pPubkey}
	return pk.VerifySignature(message, signature)
}

// VerifyP2pAddrBytes verify whether P2P address matches public key
func VerifyP2pAddrBytes(p2pPubkey []byte, p2pAddr string) bool {
	pk := ed25519.PubKeyFromBytes(p2pPubkey)
	address := P2PAddress(pk.Address())
	address2, err := P2PAddressFromBech32(p2pAddr)
	if err != nil {
		return false
	}
	return address.Equals(address2)
}

func VerifyWalletAddrBytes(walletPubkey []byte, walletAddr string) bool {
	pk := secp256k1.PubKeyFromBytes(walletPubkey)
	address := WalletAddress(pk.Address())
	address2, err := WalletAddressFromBech32(walletAddr)
	if err != nil {
		return false
	}
	return address.Equals(address2)
}

func VerifyWalletAddr(walletPubkey, walletAddr string) bool {
	pk, err := WalletPubKeyFromBech32(walletPubkey)
	if err != nil {
		return false
	}
	address := WalletAddress(pk.Address())
	address2, err := WalletAddressFromBech32(walletAddr)
	if err != nil {
		return false
	}
	return address.Equals(address2)
}

// VerifyWalletSign bech32 format of pubkey, the hex encoded signature, and the sign message
func VerifyWalletSign(walletPubkey, signature, message string) bool {
	pk, err := WalletPubKeyFromBech32(walletPubkey)
	if err != nil {
		return false
	}

	sig, err := hex.DecodeString(signature)
	if err != nil {
		utils.ErrorLog("wrong signature")
		return false
	}

	return VerifyWalletSignBytes(pk.Bytes(), sig, message)
}
