package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"github.com/stratosnet/sds/utils"
)

func EncryptAES(key, plaintext []byte, nonce uint64) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	noncePadding := make([]byte, gcm.NonceSize()-8)
	nonceFull := append(noncePadding, utils.Uint64ToBytes(nonce)...)

	ciphertext := gcm.Seal(nil, nonceFull, plaintext, nil)
	return ciphertext, nil
}

func DecryptAES(key, ciphertext []byte, nonce uint64) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	noncePadding := make([]byte, gcm.NonceSize()-8)
	nonceFull := append(noncePadding, utils.Uint64ToBytes(nonce)...)

	plaintext, err := gcm.Open(nil, nonceFull, ciphertext, nil)
	return plaintext, err
}
