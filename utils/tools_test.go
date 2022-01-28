package utils

import (
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/hd"
	"github.com/cosmos/go-bip39"
	"github.com/ipfs/go-cid"
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
	fileData := []byte("file data")
	sliceData := []byte("slice data")
	sliceNumber := uint64(1)

	fileHash := calcFileHash(fileData)
	sliceHash := CalcSliceHash(sliceData, fileHash, sliceNumber)
	fileCid, _ := cid.Decode(fileHash)
	sliceCid, _ := cid.Decode(sliceHash)
	filePrefix := fileCid.Prefix()
	slicePrefix := sliceCid.Prefix()

	expectedPrefix := cid.Prefix{
		Version:  1,
		Codec:    85,
		MhType:   27,
		MhLength: 20,
	}

	if len(fileHash) != 40 {
		t.Fatal("incorrect file hash length")
	}

	if len(sliceHash) != 40 {
		t.Fatal("incorrect slice hash length")
	}

	if filePrefix != expectedPrefix {
		t.Fatal("incorrect file cid prefix after decoding")
	}

	if slicePrefix != expectedPrefix {
		t.Fatal("incorrect slice cid prefix after decoding")
	}

	fakeFileHash := "t05ahm87h28vdd04qu3pbv0op4jnjnkpete9eposh2l6r1hp8i0hbqictcc======"
	if sliceHash == CalcSliceHash(sliceData, fakeFileHash, sliceNumber) {
		t.Fatal("slice hash should be different when being generated with different file hash")
	}
}

func TestCidLargeFile(t *testing.T) {
	encryptionTag := GetRandomString(8)
	start := time.Now()
	filehash := CalcFileHash("/home/osboxes/Downloads/ideaIU-2021.2.2.tar.gz", encryptionTag)
	elapsed := time.Since(start)
	fmt.Println(filehash)
	fmt.Println(elapsed)
}
