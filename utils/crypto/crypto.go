package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	"github.com/stratosnet/sds/utils/crypto/sha3"
	"github.com/stratosnet/sds/utils/types"
	"math/big"
	"reflect"
)

var (
	secp256k1N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	secp256k1halfN = new(big.Int).Div(secp256k1N, big.NewInt(2))
)

var errInvalidPubkey = errors.New("invalid secp256k1 public key")

// Keccak256 calculates and returns the Keccak256 hash of the input data.
func Keccak256(data ...[]byte) []byte {
	d := sha3.NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}

// Keccak256Hash calculates and returns the Keccak256 hash of the input data,
// converting it to an internal Hash data structure.
func Keccak256Hash(data ...[]byte) (h types.Hash) {
	d := sha3.NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	d.Sum(h[:0])
	return h
}

// S256 returns an instance of the secp256k1 curve.
func S256() elliptic.Curve {
	return secp256k1.S256()
}

// PubKeyToAddress calculates the wallet address from the user's private key
func PrivKeyToAddress(privKey []byte) types.Address {
	tmPrivKey := secp256k1.PrivKeyBytesToTendermint(privKey)
	tmPubKey := tmPrivKey.PubKey()
	return types.BytesToAddress(tmPubKey.Address())
}

// PubKeyToAddress calculates the wallet address from the user's public key
func PubKeyToAddress(pubKey []byte) (types.Address, error) {
	tmPubKey, err := secp256k1.PubKeyBytesToTendermint(pubKey)
	if err != nil {
		return types.Address{}, err
	}
	return types.BytesToAddress(tmPubKey.Address()), nil
}

// ToECDSAUnsafe blindly converts a binary blob to a private key. It should almost
// never be used unless you are sure the input is valid and want to avoid hitting
// errors due to bad origin encoding (0 prefixes cut off).
func ToECDSAUnsafe(d []byte) *ecdsa.PrivateKey {
	priv, _ := toECDSA(d, false)
	return priv
}

// toECDSA creates a private key with the given D value. The strict parameter
// controls whether the key's length should be enforced at the curve size or
// it can also accept legacy encodings (0 prefixes).
func toECDSA(d []byte, strict bool) (*ecdsa.PrivateKey, error) {
	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = S256()
	if strict && 8*len(d) != priv.Params().BitSize {
		return nil, fmt.Errorf("invalid length, need %d bits", priv.Params().BitSize)
	}
	priv.D = new(big.Int).SetBytes(d)

	// The priv.D must < N
	if priv.D.Cmp(secp256k1N) >= 0 {
		return nil, fmt.Errorf("invalid private key, >=N")
	}
	// The priv.D must not be zero or negative.
	if priv.D.Sign() <= 0 {
		return nil, fmt.Errorf("invalid private key, zero or negative")
	}

	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(d)
	if priv.PublicKey.X == nil {
		return nil, errors.New("invalid private key")
	}
	return priv, nil
}

// UnmarshalPubkey converts bytes to a secp256k1 public key.
func UnmarshalPubkey(pub []byte) (*ecdsa.PublicKey, error) {
	x, y := elliptic.Unmarshal(S256(), pub)
	if x == nil {
		return nil, errInvalidPubkey
	}
	return &ecdsa.PublicKey{Curve: S256(), X: x, Y: y}, nil
}

func FromECDSAPub(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(S256(), pub.X, pub.Y)
}

// MerkleTree 默克尔树处理
func MerkleTree(list interface{}) (types.Hash, error) {
	// 先将数据转成 []interface{} 的类型
	tmp := reflect.ValueOf(list)
	if tmp.Kind() != reflect.Slice {
		return types.Hash{}, errors.New("input required a slice")
	}
	data := make([]interface{}, tmp.Len())
	for i := 0; i < tmp.Len(); i++ {
		data[i] = tmp.Index(i).Interface()
	}

	if len(data) == 0 {
		return types.Hash{}, nil
	}

	hs := make([]types.Hash, len(data))
	for i, v := range data {
		b, err := json.Marshal(v)
		if err != nil {
			return types.Hash{}, nil
		}
		hs[i] = Keccak256Hash(b)
	}

	res := generateMerkleTree(hs)
	return res[0], nil
}

func generateMerkleTree(hashList []types.Hash) []types.Hash {
	l := len(hashList)
	if l == 1 {
		return hashList
	}

	if l%2 == 1 {
		hashList = append(hashList, hashList[l-1])
		l++
	}
	hs := make([]types.Hash, l/2)
	for i, j := 0, 0; i < l; i, j = i+2, j+1 {
		h := make([]byte, types.HashLength*2)
		copy(h[:types.HashLength], hashList[i].Bytes())
		copy(h[types.HashLength:], hashList[i+1].Bytes())
		hs[j] = Keccak256Hash(h)
	}

	return generateMerkleTree(hs)
}
