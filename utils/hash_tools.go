package utils

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"hash/crc32"
	"io"
	"os"

	"github.com/stratosnet/sds/utils/crypto"
)

// CalcCRC32
func CalcCRC32(data []byte) uint32 {
	iEEE := crc32.NewIEEE()
	io.WriteString(iEEE, string(data))
	return iEEE.Sum32()
}

// CalcMD5
func CalcMD5(data []byte) []byte {
	MD5 := md5.New()
	MD5.Write(data)
	MD5.Sum(nil)
	return MD5.Sum(nil)
}

// CalcFileMD5
func CalcFileMD5(filePath string) []byte {
	file, err := os.Open(filePath)
	if err != nil {
		Log(err.Error())
		return nil
	}
	defer file.Close()
	MD5 := md5.New()
	io.Copy(MD5, file)
	return MD5.Sum(nil)
}

// CalcFileCRC32
func CalcFileCRC32(filePath string) uint32 {
	file, err := os.Open(filePath)
	if err != nil {
		Log(err.Error())
		return 0
	}
	defer file.Close()
	iEEE := crc32.NewIEEE()
	io.Copy(iEEE, file)
	return iEEE.Sum32()
}

// CalcFileHash
// @notice keccak256(md5(file))
func CalcFileHash(filePath string) string {
	if filePath == "" {
		Log(errors.New("CalcFileHash: missing file path"))
		return ""
	}
	return hex.EncodeToString(crypto.Keccak256(CalcFileMD5(filePath)))
}

// CalcHash
func CalcHash(data []byte) string {
	return hex.EncodeToString(crypto.Keccak256(data))
}
