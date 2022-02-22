package utils

import (
	"bytes"
	"compress/gzip"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"math/big"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/stratosnet/sds/utils/crypto"

	"github.com/google/uuid"
)

// BytesToInt16
func BytesToInt16(b []byte) int16 {
	bytesBuffer := bytes.NewBuffer(b)

	var x int16
	binary.Read(bytesBuffer, binary.BigEndian, &x)

	return x
}

// BytesToUInt64
func BytesToUInt64(b []byte) uint64 {
	bytesBuffer := bytes.NewBuffer(b)

	var x uint64
	binary.Read(bytesBuffer, binary.BigEndian, &x)

	return x
}

// BytesToUInt32
func BytesToUInt32(b []byte) uint32 {
	bytesBuffer := bytes.NewBuffer(b)

	var x uint32
	binary.Read(bytesBuffer, binary.BigEndian, &x)

	return x
}

// BytesToUint16
func BytesToUint16(b []byte) uint16 {
	bytesBuffer := bytes.NewBuffer(b)

	var x uint16
	binary.Read(bytesBuffer, binary.BigEndian, &x)

	return x
}

// Int16ToBytes
func Int16ToBytes(n int16) []byte {
	x := n

	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

// Uint64ToBytes
func Uint64ToBytes(n uint64) []byte {
	x := n

	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

// Uint32ToBytes
func Uint32ToBytes(n uint32) []byte {
	x := n

	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

// Uint16ToBytes
func Uint16ToBytes(n uint16) []byte {
	x := n

	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

// ByteToString
func ByteToString(p []byte) string {
	// return hex.EncodeToString(p)
	for i := 0; i < len(p); i++ {
		if p[i] == 0 {
			return string(p[0:i])
		}
	}
	return string(p)
}

// Int64ToByte
func Int64ToByte(n int64) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

// MergeByte
func MergeByte(b ...[]byte) []byte {
	buf := new(bytes.Buffer)
	for _, bb := range b {
		buf.Write(bb)
	}
	return buf.Bytes()
}

func MergeBytes(a, b []byte) []byte {
	aLen := len(a)
	bLen := len(b)
	data := make([]byte, aLen+bLen)
	copy(data[:aLen], a)
	copy(data[aLen:], b)
	return data
}

func Crc32IEEE(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}

// Struct2Map
func Struct2Map(obj interface{}) map[string]interface{} {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	var data = make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		data[t.Field(i).Name] = v.Field(i).Interface()
	}
	return data
}

// ECCSign signs the given text
func ECCSign(text []byte, prk *ecdsa.PrivateKey) ([]byte, error) {
	randSign := CalcHash([]byte(uuid.New().String() + "#" + strconv.FormatInt(time.Now().UnixNano(), 10)))
	r, s, err := ecdsa.Sign(strings.NewReader(randSign), prk, text)
	if err != nil {
		return nil, err
	}
	rt, err := r.MarshalText()
	if err != nil {
		return nil, err
	}
	st, err := s.MarshalText()
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	defer w.Close()
	_, err = w.Write([]byte(string(rt) + "+" + string(st)))
	if err != nil {
		return nil, err
	}
	w.Flush()
	return b.Bytes(), nil
}

// ECCSignBytes converts the private key bytes to an ecdsa.PrivateKey and then signs the given text
func ECCSignBytes(text, privateKey []byte) ([]byte, error) {
	privKey, _ := btcec.PrivKeyFromBytes(crypto.S256(), privateKey)
	return ECCSign(text, privKey.ToECDSA())
}

// ECCVerify verifies the given signature
func ECCVerify(text []byte, signature []byte, key *ecdsa.PublicKey) bool {

	r, err := gzip.NewReader(bytes.NewBuffer(signature))
	if err != nil {
		ErrorLog(errors.New("decode error," + err.Error()))
		return false
	}
	defer r.Close()
	buf := make([]byte, 1024)
	count, err := r.Read(buf)
	if err != nil {
		ErrorLog(errors.New("decode read error," + err.Error()))
		return false
	}
	rs := strings.Split(string(buf[:count]), "+")
	if len(rs) != 2 {
		ErrorLog(errors.New("decode fail"))
		return false
	}

	var rint big.Int
	var sint big.Int

	err = rint.UnmarshalText([]byte(rs[0]))
	if err != nil {
		ErrorLog(errors.New("decrypt rint fail, " + err.Error()))
		return false
	}
	err = sint.UnmarshalText([]byte(rs[1]))
	if err != nil {
		ErrorLog(errors.New("decrypt sint fail, " + err.Error()))
		return false
	}

	return ecdsa.Verify(key, text, &rint, &sint)
}

// ECCVerifyBytes converts the public key bytes to an ecdsa.PublicKey and then verifies the given signature
func ECCVerifyBytes(text, signature, publicKey []byte) bool {
	pubKey, err := crypto.UnmarshalPubkey(publicKey)
	if err != nil {
		return false
	}
	return ECCVerify(text, signature, pubKey)
}

// CheckStructField
func CheckStructField(field string, structName interface{}) bool {
	t := reflect.TypeOf(structName)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		ErrorLog("Check type error not Struct")
		return false
	}
	fieldNum := t.NumField()
	for i := 0; i < fieldNum; i++ {
		tagName := t.Field(i).Name
		if tagName == field {
			//DebugLog("include field: " + field)
			return true
		}
	}
	return false
}

// IntToString
func IntToString(i int) string {
	return strconv.Itoa(i)
}

// StringToInt
func StringToInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func Absolute(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return filepath.Abs(path)
	}
	return path, nil
}
