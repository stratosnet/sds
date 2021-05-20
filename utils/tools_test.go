package utils

import (
	"github.com/cosmos/cosmos-sdk/crypto/keys/hd"
	"github.com/cosmos/go-bip39"
	"github.com/stratosnet/sds/utils/crypto"
	"github.com/stratosnet/sds/utils/crypto/math"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	"testing"
)

func TestECCSignAndVerify(t *testing.T) {
	mnemonic := "vacant cool enlist kiss van despair ethics silly route master funny door gossip athlete sword language argue alien any item desk mystery tray parade"
	pass := ""
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, pass)
	if err != nil {
		t.Fatal("couldn't generate seed: " + err.Error())
	}
	masterPriv, ch := hd.ComputeMastersFromSeed(seed)
	hdPath := "44'/606'/0'/0/0"
	derivedKey, err := hd.DerivePrivateKeyForPath(masterPriv, ch, hdPath)
	privateKeyECDSA := crypto.ToECDSAUnsafe(derivedKey[:])

	publicKeyECDSA := &privateKeyECDSA.PublicKey
	privKeyBytes := math.PaddedBigBytes(privateKeyECDSA.D, 32)
	pubKeyBytes := secp256k1.PrivKeyToPubKey(privKeyBytes)

	msg := []byte("this is a random message")
	sig1, err := ECCSign(msg, privateKeyECDSA)
	if err != nil {
		t.Fatal("couldn't sign with ecdsa.PrivateKey: " + err.Error())
	}
	sig2, err := ECCSignBytes(msg, privKeyBytes)
	if err != nil {
		t.Fatal("couldn't sign with bytes: " + err.Error())
	}

	if !ECCVerify(msg, sig1, publicKeyECDSA) {
		t.Fatal("couldn't ECCVerify sig from ECCSign")
	}
	if !ECCVerify(msg, sig2, publicKeyECDSA) {
		t.Fatal("couldn't ECCVerify sig from ECCSignBytes")
	}
	if !ECCVerifyBytes(msg, sig1, pubKeyBytes) {
		t.Fatal("couldn't ECCVerifyBytes sig from ECCSign")
	}
	if !ECCVerifyBytes(msg, sig2, pubKeyBytes) {
		t.Fatal("couldn't ECCVerifyBytes sig from ECCSignBytes")
	}
}
