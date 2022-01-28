package utils

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/hd"
	"github.com/cosmos/go-bip39"
	"github.com/stratosnet/sds/utils/crypto"
	"github.com/stratosnet/sds/utils/crypto/math"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
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
		t.Fatal("couldn't sign with ecdsa.P2PPrivateKey: " + err.Error())
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

func TestCid(t *testing.T) {
	MyLogger.SetLogLevel(Error)
	for i := 0; i < 100; i++ {
		var fileData [256]byte
		_, err := rand.Read(fileData[:])
		if err != nil {
			t.Fatal("cannot generate random data", err)
		}

		var sliceData [256]byte
		_, err = rand.Read(sliceData[:])
		if err != nil {
			t.Fatal("cannot generate random data", err)
		}

		fileHash := calcFileHash(fileData[:])
		sliceHash := CalcSliceHash(sliceData[:], fileHash, uint64(i))

		if !VerifyHash(fileHash) {
			t.Fatal("generated file hash is invalid")
		}

		if !VerifyHash(sliceHash) {
			t.Fatal("generated slice hash is invalid")
		}

		fakeFileHash := "t05ahm87h28vdd04qu3pbv0op4jnjnkpete9eposh2l6r1hp8i0hbqictcc======"
		if sliceHash == CalcSliceHash(sliceData[:], fakeFileHash, uint64(i)) {
			t.Fatal("slice hash should be different when being generated with different file hash")
		}

		if VerifyHash(fakeFileHash) {
			t.Fatal("Fake file hash should have failed verification")
		}
	}
}

func TestCidLargeFile(t *testing.T) {
	encryptionTag := GetRandomString(8)
	start := time.Now()
	filehash := CalcFileHash("/home/osboxes/Downloads/ideaIU-2021.2.2.tar.gz", encryptionTag)
	elapsed := time.Since(start)
	fmt.Println(filehash)
	fmt.Println(elapsed)

	if !VerifyHash(filehash) {
		t.Fatal("generated file hash is invalid")
	}
}
