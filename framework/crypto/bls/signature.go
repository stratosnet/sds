package bls

import (
	"math/big"

	"github.com/Nik-U/pbc"
	"github.com/pkg/errors"
)

var (
	pairing   *pbc.Pairing
	generator *pbc.Element
)

func init() {
	err := loadPairing()
	if err != nil {
		panic(err)
	}

	err = loadGenerator()
	if err != nil {
		panic(err)
	}
}

// loadPairing Loads the pairing used by the pbc library. Must be called before signatures can be created or verified.
func loadPairing() error {
	params, err := pbc.NewParamsFromString(blsParams)
	if err != nil {
		return err
	}

	pairing = params.NewPairing()
	return nil
}

// loadGenerator loads the generator params for the pbc library. Must be called after LoadPairing, and before other methods
func loadGenerator() error {
	if pairing == nil {
		return errors.New("The BLS pairing hasn't been initialized yet")
	}

	generatorValue, success := pairing.NewG2().SetString(blsGenerator, 0)
	if generatorValue == nil || !success {
		return errors.New("Invalid generator params for BLS signature")
	}

	generator = generatorValue
	return nil
}

// Sign signs some data using a BLS private key
func Sign(data, privateKey []byte) (signature []byte, err error) {
	if err := verifyInitialization(); err != nil {
		return nil, err
	}

	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("BLS signature error, low-level cgocall signal error: %v", r)
		}
	}()

	h := pairing.NewG1().SetFromHash(data)
	privateKeyElement := pairing.NewZr().SetBig(big.NewInt(0).SetBytes(privateKey))
	signatureElement := pairing.NewG2().ThenMul(pairing.NewG2().PowZn(h, privateKeyElement))

	return signatureElement.CompressedBytes(), nil
}

// SignAndAggregate signs the data using the privateKey, then aggregate the signature with an existing signature
func SignAndAggregate(data, privateKey, existingSignature []byte) (signature []byte, err error) {
	if err := verifyInitialization(); err != nil {
		return nil, err
	}

	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("BLS signature error, low-level cgocall signal error: %v", r)
		}
	}()

	existingSignatureElement := pairing.NewG2().SetCompressedBytes(existingSignature)

	h := pairing.NewG1().SetFromHash(data)
	privateKeyElement := pairing.NewZr().SetBig(big.NewInt(0).SetBytes(privateKey))
	signatureElement := existingSignatureElement.ThenMul(pairing.NewG2().PowZn(h, privateKeyElement))

	return signatureElement.CompressedBytes(), nil
}

// AggregateSignatures aggregates multiple signatures together
func AggregateSignatures(signatures ...[]byte) (aggregatedSignature []byte, err error) {
	if err := verifyInitialization(); err != nil {
		return nil, err
	}

	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("BLS signature error, low-level cgocall signal error: %v", r)
		}
	}()

	aggregatedSignatureElement := pairing.NewG2()
	for _, signature := range signatures {
		signatureElement := pairing.NewG2().SetCompressedBytes(signature)
		aggregatedSignatureElement = aggregatedSignatureElement.ThenMul(signatureElement)
	}

	return aggregatedSignatureElement.CompressedBytes(), nil
}

// Verify verifies a BLS signature. It requires a public key for each private key that was used in generating the signature
func Verify(data, signature []byte, pubKeys ...[]byte) (result bool, err error) {
	if err := verifyInitialization(); err != nil {
		return false, err
	}

	if len(pubKeys) < 1 {
		return false, errors.New("BLS signatures cannot be verified without providing a public key")
	}

	defer func() {
		if r := recover(); r != nil {
			result = false
			err = errors.Errorf("BLS verification error, low-level cgocall signal error: %v", r)
		}
	}()

	var combinedPubKey *pbc.Element
	for _, pubKey := range pubKeys {
		pubKeyElement := pairing.NewG2().SetCompressedBytes(pubKey)
		if combinedPubKey == nil {
			combinedPubKey = pubKeyElement
		} else {
			combinedPubKey = combinedPubKey.ThenMul(pubKeyElement)
		}
	}

	h := pairing.NewG1().SetFromHash(data)
	signatureElement := pairing.NewG1().SetCompressedBytes(signature)

	tmp1 := pairing.NewGT().Pair(h, combinedPubKey)
	tmp2 := pairing.NewGT().Pair(signatureElement, generator)
	return tmp1.Equals(tmp2), nil
}

func verifyInitialization() error {
	if pairing == nil {
		return errors.New("The BLS pairing hasn't been initialized yet")
	}

	if generator == nil {
		return errors.New("The BLS generator params haven't been initialized yet")
	}
	return nil
}
