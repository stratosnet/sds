package utils

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"strconv"
)

const (
	SIZE_OF_INT8   = 1
	SIZE_OF_UINT8  = 1
	SIZE_OF_INT16  = 2 // in byte
	SIZE_OF_UINT16 = 2 // in byte
	SIZE_OF_INT32  = 4 // in byte
	SIZE_OF_UINT32 = 4 // in byte
	SIZE_OF_INT64  = 8 // in byte
	SIZE_OF_UINT64 = 8 // in byte
)

func BytesToInt16(b []byte) int16 {
	bytesBuffer := bytes.NewBuffer(b)

	var x int16
	_ = binary.Read(bytesBuffer, binary.BigEndian, &x)

	return x
}

func BytesToInt64(b []byte) int64 {
	bytesBuffer := bytes.NewBuffer(b)

	var x int64
	_ = binary.Read(bytesBuffer, binary.BigEndian, &x)

	return x
}

func BytesToUInt64(b []byte) uint64 {
	bytesBuffer := bytes.NewBuffer(b)

	var x uint64
	_ = binary.Read(bytesBuffer, binary.BigEndian, &x)

	return x
}

func BytesToUInt32(b []byte) uint32 {
	bytesBuffer := bytes.NewBuffer(b)

	var x uint32
	_ = binary.Read(bytesBuffer, binary.BigEndian, &x)

	return x
}

func BytesToUint16(b []byte) uint16 {
	bytesBuffer := bytes.NewBuffer(b)

	var x uint16
	_ = binary.Read(bytesBuffer, binary.BigEndian, &x)

	return x
}

func Int16ToBytes(n int16) []byte {
	x := n

	bytesBuffer := bytes.NewBuffer([]byte{})
	_ = binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

func Uint64ToBytes(n uint64) []byte {
	x := n

	bytesBuffer := bytes.NewBuffer([]byte{})
	_ = binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

func Uint32ToBytes(n uint32) []byte {
	x := n

	bytesBuffer := bytes.NewBuffer([]byte{})
	_ = binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

func Uint16ToBytes(n uint16) []byte {
	x := n

	bytesBuffer := bytes.NewBuffer([]byte{})
	_ = binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

func Uint8ToBytes(n uint8) []byte {
	x := n

	bytesBuffer := bytes.NewBuffer([]byte{})
	_ = binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

func ByteToString(p []byte) string {
	// return hex.EncodeToString(p)
	for i := 0; i < len(p); i++ {
		if p[i] == 0 {
			return string(p[0:i])
		}
	}
	return string(p)
}

func Int64ToByte(n int64) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	_ = binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

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

func IntToString(i int) string {
	return strconv.Itoa(i)
}

func StringToInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func Absolute(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return filepath.Abs(path)
	}
	return path, nil
}
