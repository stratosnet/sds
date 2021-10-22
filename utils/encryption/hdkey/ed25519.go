package hdkey

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"github.com/btcsuite/btcutil"
)

type Ed25519 struct{}

var (
	ErrNonHardenedChild       = errors.New("ed25519 does not support non hardened children")
	ErrPublicParentDerivation = errors.New("derivation from public parent not supported")
)

func (e *Ed25519) Child(k ExtendedKey, i uint32) (*ExtendedKey, error) {
	// Prevent derivation of children beyond the max allowed depth.
	if k.depth == maxUint8 {
		return nil, ErrDeriveBeyondMaxDepth
	}

	// There are four scenarios that could happen here:
	// 1) Private extended key -> Hardened child private extended key
	// 2) Private extended key -> Non-hardened child private extended key
	// 3) Public extended key -> Non-hardened child public extended key
	// 4) Public extended key -> Hardened child public extended key (INVALID!)

	// Case #4 is invalid, so error out early.
	// A hardened child extended key may not be created from a public
	// extended key.
	isChildHardened := i >= HardenedKeyStart
	if !k.isPrivate && isChildHardened {
		return nil, ErrDeriveHardFromPublic
	}

	// The data used to derive the child key depends on whether or not the
	// child is hardened per [BIP32].
	//
	// For hardened children:
	//   0x00 || ser256(parentKey) || ser32(i)
	//
	// For normal children:
	//   serP(parentPubKey) || ser32(i)
	keyLen := 33
	data := make([]byte, keyLen+4)
	if isChildHardened {
		// Case #1.
		// When the child is a hardened child, the key is known to be a
		// private key due to the above early return.  Pad it with a
		// leading zero as required by [BIP32] for deriving the child.
		copy(data[1:], k.key)
	} else {
		// Case #2 or #3.
		return nil, ErrNonHardenedChild
	}
	binary.BigEndian.PutUint32(data[keyLen:], i)

	// Take the HMAC-SHA512 of the current key's chain code and the derived
	// data:
	//   I = HMAC-SHA512(Key = chainCode, Data = data)
	hmac512 := hmac.New(sha512.New, k.chainCode)
	hmac512.Write(data)
	ilr := hmac512.Sum(nil)

	// Split "I" into two 32-byte sequences Il and Ir where:
	//   Il = intermediate key used to derive the child
	//   Ir = child chain code
	il := ilr[:len(ilr)/2]
	childChainCode := ilr[len(ilr)/2:]
	childKey := il
	isPrivate := true

	// The fingerprint of the parent for the derived child is the first 4
	// bytes of the RIPEMD160(SHA256(parentPubKey)).
	parentFP := btcutil.Hash160(pubKeyBytes(&k))[:4]
	return NewExtendedKey(childKey, childChainCode, parentFP,
		k.depth+1, i, isPrivate), nil
}

func Ed25519Child(k *ExtendedKey, i uint32) (*ExtendedKey, error) {
	if k == nil {
		return nil, errors.New("the given key pointer was nil")
	}
	gen := Ed25519{}
	return gen.Child(*k, i+HardenedKeyStart)
}

func Ed25519Child64(k *ExtendedKey, i uint64) (*ExtendedKey, error) {
	highBytes := uint32(i >> 42)
	middleBytes := uint32((i << 22) >> 43)
	lowBytes := uint32((i << 43) >> 43)

	child1, err := Ed25519Child(k, highBytes)
	if err != nil {
		return nil, err
	}

	child2, err := Ed25519Child(child1, middleBytes)
	if err != nil {
		return nil, err
	}

	return Ed25519Child(child2, lowBytes)
}
