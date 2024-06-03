package encryption

import (
	"bytes"
	"crypto/ed25519"
	"testing"
)

func TestECDH(t *testing.T) {
	publicA, privateA, _ := ed25519.GenerateKey(nil)
	publicB, privateB, _ := ed25519.GenerateKey(nil)

	shared1, _ := ECDH(privateA, publicB)
	shared2, _ := ECDH(privateB, publicA)
	if !bytes.Equal(shared1, shared2) {
		t.Fatal("Both shared secrets should be equal")
	}
}
