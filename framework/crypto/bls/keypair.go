package bls

import (
	"encoding/json"
	"io/ioutil"
)

type KeyPair struct {
	PublicKey  []byte `json:"publicKey"`
	PrivateKey []byte `json:"privateKey"`
}

// NewKeyPair creates a new BLS keypair
func NewKeyPair() (privateKey, publicKey []byte, err error) {
	if err = verifyInitialization(); err != nil {
		return nil, nil, err
	}

	privateKeyElement := pairing.NewZr().Rand()
	privateKey = privateKeyElement.BigInt().Bytes()

	publicKeyElement := pairing.NewG2().PowZn(generator, privateKeyElement)
	publicKey = publicKeyElement.CompressedBytes()
	return
}

// NewKeyPairFromBytes creates a new BLS keypair deterministically based on a given seed
func NewKeyPairFromBytes(seed []byte) (privateKey, publicKey []byte, err error) {
	if err = verifyInitialization(); err != nil {
		return nil, nil, err
	}

	privateKeyElement := pairing.NewZr().SetFromHash(seed)
	privateKey = privateKeyElement.BigInt().Bytes()

	publicKeyElement := pairing.NewG2().PowZn(generator, privateKeyElement)
	publicKey = publicKeyElement.CompressedBytes()
	return
}

// StoreKeyPair stores a BLS keypair to a file
func StoreKeyPair(privateKey, publicKey []byte, fileLocation string) error {
	keyPair := KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}
	keyPairJson, err := json.Marshal(keyPair)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fileLocation, keyPairJson, 0666)
}

// LoadKeyPair loads a BLS keypair from a file
func LoadKeyPair(keyPairLocation string) (privateKey, publicKey []byte, err error) {
	keyPairJson, err := ioutil.ReadFile(keyPairLocation)
	if err != nil {
		return nil, nil, err
	}
	keyPair := &KeyPair{}
	err = json.Unmarshal(keyPairJson, keyPair)
	if err != nil {
		return nil, nil, err
	}

	return keyPair.PrivateKey, keyPair.PublicKey, nil
}
