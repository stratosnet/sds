package encryption

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestAESEncryption(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatal("Couldn't generate key: " + err.Error())
	}
	message := make([]byte, 2<<20)
	_, err = rand.Read(message)
	if err != nil {
		t.Fatal("Couldn't generate random message" + err.Error())
	}

	wrongKey := make([]byte, 32)
	copy(wrongKey, key)
	wrongKey[0] = ^wrongKey[0]

	ciphertext, err := EncryptAES(key, message, 42)
	if err != nil {
		t.Fatal("Couldn't encrypt message" + err.Error())
	}
	wrongKeyCiphertext, err := EncryptAES(wrongKey, message, 42)
	if err != nil {
		t.Fatal("Couldn't encrypt message" + err.Error())
	}
	wrongNonceCiphertext, err := EncryptAES(key, message, 666)
	if err != nil {
		t.Fatal("Couldn't encrypt message" + err.Error())
	}

	if bytes.Equal(ciphertext, wrongKeyCiphertext) || bytes.Equal(ciphertext, wrongNonceCiphertext) {
		t.Fatal("Using a different key or nonce should generate a different ciphertext")
	}
	if bytes.Equal(message, ciphertext) {
		t.Fatal("The encrypted message should not equl the original message")
	}

	plaintext, err := DecryptAES(key, ciphertext, 42)
	if err != nil {
		t.Fatal("Couldn't decrypt message" + err.Error())
	}
	_, err = DecryptAES(wrongKey, ciphertext, 42)
	if err == nil {
		t.Fatal("Using the wrong key should make it impossible to decrypt the message")
	}
	_, err = DecryptAES(key, ciphertext, 666)
	if err == nil {
		t.Fatal("Using the wrong nonce should make it impossible to decrypt the message")
	}

	if !bytes.Equal(message, plaintext) {
		t.Fatal("The decrypted message should equal the original message")
	}
}
