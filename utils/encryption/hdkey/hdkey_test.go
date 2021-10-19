package hdkey

import (
	"bytes"
	"crypto/ed25519"
	"os"
	"reflect"
	"testing"
)

func TestHDKeyMnemonic(t *testing.T) {
	err := os.Chdir("../../..") // Testing changes the current directory. We need the current directory to be the project root for the mnemonic library to work
	if err != nil {
		t.Fatal(err)
	}
	passphrase := "test1"
	key, mnemonic, err := MasterKeyFromPassphrase(passphrase)
	if err != nil {
		t.Fatal(err)
	}
	if len(mnemonic) < 24 {
		t.Fatal("Not enough mnemonic")
	}

	_, mnemonic2, err := MasterKeyFromPassphrase(passphrase)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(mnemonic, mnemonic2) {
		t.Fatal("Two master keys using the same passphrase should still yield different mnemonic code")
	}

	key3, err := MasterKeyFromMnemonic(mnemonic, passphrase)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(key.PrivateKey(), key3.PrivateKey()) || !bytes.Equal(key.PublicKey(), key3.PublicKey()) {
		t.Fatal("Cannot recover master key from mnemonic")
	}
}

func TestHDKeyChild(t *testing.T) {
	err := os.Chdir("../../..") // Testing changes the current directory. We need the current directory to be the project root for the mnemonic library to work
	if err != nil {
		t.Fatal(err)
	}
	passphrase := "test1"
	key, mnemonic, err := MasterKeyFromPassphrase(passphrase)
	if err != nil {
		t.Fatal(err)
	}

	childKey, err := Ed25519Child(key, 42)
	if err != nil {
		t.Fatal("Couldn't generate a child key", err)
	}
	message := "some data to sign"
	messageBytes := []byte(message)
	fullChildKey := ed25519.NewKeyFromSeed(childKey.PrivateKey())
	signature := ed25519.Sign(fullChildKey, messageBytes)

	key2, err := MasterKeyFromMnemonic(mnemonic, passphrase)
	childKey2, err := Ed25519Child(key2, 42)
	if err != nil {
		t.Fatal("Couldn't generate a child key", err)
	}
	if !ed25519.Verify(childKey2.PublicKey(), messageBytes, signature) {
		t.Fatal("Generating the same child twice didn't return the same keys")
	}
}

func TestHDKeySignature(t *testing.T) {
	err := os.Chdir("../../..") // Testing changes the current directory. We need the current directory to be the project root for the mnemonic library to work
	if err != nil {
		t.Fatal(err)
	}
	passphrase := "test1"
	key, _, err := MasterKeyFromPassphrase(passphrase)
	if err != nil {
		t.Fatal(err)
	}

	message := "some data to sign"
	messageBytes := []byte(message)
	fullKey := ed25519.NewKeyFromSeed(key.PrivateKey())
	signature := ed25519.Sign(fullKey, messageBytes)
	if !ed25519.Verify(key.PublicKey(), messageBytes, signature) {
		t.Fatal("The master key couldn't be used as an Ed25519 key")
	}
}

func TestEd25519Child64(t *testing.T) {
	err := os.Chdir("../../..") // Testing changes the current directory. We need the current directory to be the project root for the mnemonic library to work
	if err != nil {
		t.Fatal(err)
	}
	passphrase := "test1"
	key, _, err := MasterKeyFromPassphrase(passphrase)
	if err != nil {
		t.Fatal(err)
	}

	var highBytes uint32 = 123456
	var middleBytes uint32 = 987654
	var lowBytes uint32 = 112233
	full := uint64(highBytes) << 42
	full += uint64(middleBytes) << 21
	full += uint64(lowBytes)

	child1, err := Ed25519Child(key, highBytes)
	if err != nil {
		t.Fatal(err)
	}
	child2, err := Ed25519Child(child1, middleBytes)
	if err != nil {
		t.Fatal(err)
	}
	child3, err := Ed25519Child(child2, lowBytes)
	if err != nil {
		t.Fatal(err)
	}
	childFull, err := Ed25519Child64(key, full)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(child3.PrivateKey(), childFull.PrivateKey()) {
		t.Fatal("Ed25519Child64 doesn't return the expected result")
	}
}
