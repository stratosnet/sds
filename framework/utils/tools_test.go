package utils

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"math/big"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/cosmos/go-bip39"
	"github.com/ipfs/go-cid"
	mbase "github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"

	"github.com/stratosnet/sds/framework/utils/crypto"
	"github.com/stratosnet/sds/framework/utils/crypto/math"
)

func init() {
	NewDefaultLogger("", false, false)
}

func TestECCSignAndVerify(t *testing.T) {
	mnemonic := "vacant cool enlist kiss van despair ethics silly route master funny door gossip athlete sword language argue alien any item desk mystery tray parade"
	pass := ""
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, pass)
	if err != nil {
		t.Fatal("couldn't generate seed: " + err.Error())
	}
	masterPriv, ch := ComputeMastersFromSeed(seed)
	hdPath := "44'/606'/0'/0/0"
	derivedKey, err := DerivePrivateKeyForPath(masterPriv, ch, hdPath)
	if err != nil {
		t.Fatal("couldn't derive private key from seed: " + err.Error())
	}
	privateKeyECDSA := crypto.ToECDSAUnsafe(derivedKey[:])
	privKeyBytes := math.PaddedBigBytes(privateKeyECDSA.D, 32)

	publicKeyECDSA := &privateKeyECDSA.PublicKey
	pubKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)

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

		filehash, _ := mh.Sum(fileData[:], mh.KECCAK_256, 20)
		fileCid := cid.NewCidV1(uint64(SDS_CODEC), filehash)
		encoder, _ := mbase.NewEncoder(mbase.Base32hex)
		fh := fileCid.Encode(encoder)

		sliceHash := CalcSliceHash(sliceData[:], fh, uint64(i))

		if !ValidateHash(fh) {
			t.Fatal("generated file hash is invalid")
		}
		if !ValidateHash(sliceHash) {
			t.Fatal("generated slice hash is invalid")
		}

		fakeFileHash := "t05ahm87h28vdd04qu3pbv0op4jnjnkpete9eposh2l6r1hp8i0hbqictcc======"
		if sliceHash == CalcSliceHash(sliceData[:], fakeFileHash, uint64(i)) {
			t.Fatal("slice hash should be different when being generated with different file hash")
		}

		if ValidateHash(fakeFileHash) {
			t.Fatal("Fake file hash should have failed verification")
		}
	}
}

func TestCidLargeFile(t *testing.T) {
	encryptionTag := GetRandomString(8)
	fileContent := make([]byte, 1024*1024*1024) // 1GB
	rand.Read(fileContent)

	start := time.Now()
	data := append([]byte(encryptionTag), md5.New().Sum(fileContent)...)
	filehash, _ := mh.Sum(data, mh.KECCAK_256, 20)
	fileCid := cid.NewCidV1(uint64(cid.Raw), filehash)
	encoder, _ := mbase.NewEncoder(mbase.Base32hex)
	fh := fileCid.Encode(encoder)
	elapsed := time.Since(start)
	t.Log(filehash)
	t.Log(elapsed)

	if !ValidateHash(fh) {
		t.Fatal("generated file hash is invalid")
	}
}

//-----------------------------------------------------------------------------------------

// DerivePrivateKeyForPath derives the private key by following the BIP 32/44 path from privKeyBytes,
// using the given chainCode.
func DerivePrivateKeyForPath(privKeyBytes, chainCode [32]byte, path string) ([]byte, error) {
	// First step is to trim the right end path separator lest we panic.
	// See issue https://github.com/cosmos/cosmos-sdk/issues/8557
	path = strings.TrimRightFunc(path, func(r rune) bool { return r == filepath.Separator })
	data := privKeyBytes
	parts := strings.Split(path, "/")

	switch {
	case parts[0] == path:
		return nil, fmt.Errorf("path '%s' doesn't contain '/' separators", path)
	case strings.TrimSpace(parts[0]) == "m":
		parts = parts[1:]
	}

	for i, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("path %q with split element #%d is an empty string", part, i)
		}
		// do we have an apostrophe?
		harden := part[len(part)-1:] == "'"
		// harden == private derivation, else public derivation:
		if harden {
			part = part[:len(part)-1]
		}

		// As per the extended keys specification in
		// https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki#extended-keys
		// index values are in the range [0, 1<<31-1] aka [0, max(int32)]
		idx, err := strconv.ParseUint(part, 10, 31)
		if err != nil {
			return []byte{}, fmt.Errorf("invalid BIP 32 path %s: %w", path, err)
		}

		data, chainCode = derivePrivateKey(data, chainCode, uint32(idx), harden)
	}

	derivedKey := make([]byte, 32)
	n := copy(derivedKey, data[:])

	if n != 32 || len(data) != 32 {
		return []byte{}, fmt.Errorf("expected a key of length 32, got length: %d", len(data))
	}

	return derivedKey, nil
}

// derivePrivateKey derives the private key with index and chainCode.
// If harden is true, the derivation is 'hardened'.
// It returns the new private key and new chain code.
// For more information on hardened keys see:
//   - https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki
func derivePrivateKey(privKeyBytes [32]byte, chainCode [32]byte, index uint32, harden bool) ([32]byte, [32]byte) {
	var data []byte

	if harden {
		index |= 0x80000000

		data = append([]byte{byte(0)}, privKeyBytes[:]...)
	} else {
		// this can't return an error:
		_, ecPub := btcec.PrivKeyFromBytes(privKeyBytes[:])
		pubkeyBytes := ecPub.SerializeCompressed()
		data = pubkeyBytes

		/* By using btcec, we can remove the dependency on tendermint/crypto/secp256k1
		pubkey := secp256k1.PrivKeySecp256k1(privKeyBytes).PubKey()
		public := pubkey.(secp256k1.PubKeySecp256k1)
		data = public[:]
		*/
	}

	data = append(data, uint32ToBytes(index)...)
	data2, chainCode2 := i64(chainCode[:], data)
	x := addScalars(privKeyBytes[:], data2[:])

	return x, chainCode2
}

func uint32ToBytes(i uint32) []byte {
	b := [4]byte{}
	binary.BigEndian.PutUint32(b[:], i)

	return b[:]
}

// modular big endian addition
func addScalars(a []byte, b []byte) [32]byte {
	aInt := new(big.Int).SetBytes(a)
	bInt := new(big.Int).SetBytes(b)
	sInt := new(big.Int).Add(aInt, bInt)
	x := sInt.Mod(sInt, btcec.S256().N).Bytes()
	x2 := [32]byte{}
	copy(x2[32-len(x):], x)

	return x2
}

// ComputeMastersFromSeed returns the master secret key's, and chain code.
func ComputeMastersFromSeed(seed []byte) (secret [32]byte, chainCode [32]byte) {
	curveIdentifier := []byte("Bitcoin seed")
	secret, chainCode = i64(curveIdentifier, seed)

	return
}

// i64 returns the two halfs of the SHA512 HMAC of key and data.
func i64(key []byte, data []byte) (il [32]byte, ir [32]byte) {
	mac := hmac.New(sha512.New, key)
	// sha512 does not err
	_, _ = mac.Write(data)

	I := mac.Sum(nil)
	copy(il[:], I[:32])
	copy(ir[:], I[32:])

	return
}
