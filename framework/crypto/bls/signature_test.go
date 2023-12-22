package bls

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"

	fwed25519 "github.com/stratosnet/sds/framework/crypto/ed25519"
)

func init() {
	if err := os.Chdir("../.."); err != nil {
		panic(err)
	}
}

func TestSignature(t *testing.T) {
	priv, pub, err := NewKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Private key %v  Public key %v", hex.EncodeToString(priv), hex.EncodeToString(pub))

	data := []byte("this is some kind of data")
	signature, err := Sign(data, priv)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Signature " + hex.EncodeToString(signature))

	verified, err := Verify(data, signature, pub)
	if err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Fatal("couldn't verify signature")
	}

	wrongData := []byte("this is not the same data")
	verified, err = Verify(wrongData, signature, pub)
	if err != nil {
		t.Fatal(err)
	}
	if verified {
		t.Fatal("couldn't verify signature")
	}
}

func TestSignAndAggregate(t *testing.T) {
	priv, pub, err := NewKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	priv2, pub2, err := NewKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	data := []byte("this is some kind of data")
	signature, err := Sign(data, priv)
	if err != nil {
		t.Fatal(err)
	}
	signature2, err := SignAndAggregate(data, priv2, signature)
	if err != nil {
		t.Fatal(err)
	}

	verified, err := Verify(data, signature2, pub)
	if err != nil {
		t.Fatal(err)
	}
	if verified {
		t.Fatal("verify should have failed")
	}

	verified, err = Verify(data, signature2, pub, pub2)
	if err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Fatal("couldn't verify signature")
	}
}

func TestStoreLoadKeyPair(t *testing.T) {
	privKey, pubKey, err := NewKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	location := "priv_bls_key.json"
	_ = os.Remove(location)
	err = StoreKeyPair(privKey, pubKey, location)
	if err != nil {
		t.Fatal(err)
	}

	privKey2, pubKey2, err := LoadKeyPair(location)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(privKey, privKey2) {
		t.Fatal("private key is not the same after storing and loading")
	}
	if !bytes.Equal(pubKey, pubKey2) {
		t.Fatal("public key is not the same after storing and loading")
	}

	_ = os.Remove(location) // Comment this out to keep the generated key pair after testing
}

func TestSignatureFromEd25519PrivKey(t *testing.T) {
	for i := 0; i < 100; i++ {
		seed := fwed25519.GenPrivKey().Bytes()
		priv, pub, err := NewKeyPairFromBytes(seed)
		if err != nil {
			t.Fatal(err)
		}
		//t.Logf("Private key %v  Public key %v\n", hex.EncodeToString(priv), hex.EncodeToString(pub))

		data := []byte("this is some kind of data")
		signature, err := Sign(data, priv)
		if err != nil {
			t.Fatal(err)
		}
		//t.Log("Signature " + hex.EncodeToString(signature))

		verified, err := Verify(data, signature, pub)
		if err != nil {
			t.Fatal(err)
		}
		if !verified {
			t.Fatal("couldn't verify signature")
		}
	}
}
