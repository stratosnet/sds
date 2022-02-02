package utils

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash/crc32"
	"io"
	"os"

	"github.com/ipfs/go-cid"
	mbase "github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"

	"github.com/stratosnet/sds/utils/crypto"
)

const hashLen = 20

var hashCidPrefix = cid.Prefix{
	Version:  1,
	Codec:    85,
	MhType:   27,
	MhLength: 20,
}

// CalcCRC32
func CalcCRC32(data []byte) uint32 {
	iEEE := crc32.NewIEEE()
	io.WriteString(iEEE, string(data))
	return iEEE.Sum32()
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
func CalcFileHash(filePath, encryptionTag string) string {
	if filePath == "" {
		Log(errors.New("CalcFileHash: missing file path"))
		return ""
	}
	data := append([]byte(encryptionTag), CalcFileMD5(filePath)...)
	return calcFileHash(data)
}

// CalcHash
func CalcHash(data []byte) string {
	return hex.EncodeToString(crypto.Keccak256(data))
}

// CalcHash
func CalcSliceHash(data []byte, fileHash string, sliceNumber uint64) string {
	fileCid, _ := cid.Decode(fileHash)
	fileKeccak256 := fileCid.Hash()
	sliceNumBytes := uint64ToBytes(sliceNumber)
	data = append(sliceNumBytes, data...)
	sliceKeccak256, _ := mh.Sum(data, mh.KECCAK_256, hashLen)
	if len(fileKeccak256) != len(sliceKeccak256) {
		Log(errors.New("length of fileKeccak256 and sliceKeccak256 doesn't match"))
		return ""
	}
	sliceHash := make([]byte, len(fileKeccak256))
	for i := 0; i < len(fileKeccak256); i++ {
		sliceHash[i] = fileKeccak256[i] ^ sliceKeccak256[i]
	}
	sliceHash, _ = mh.Sum(sliceHash, mh.KECCAK_256, hashLen)
	sliceCid := cid.NewCidV1(cid.Raw, sliceHash)
	encoder, _ := mbase.NewEncoder(mbase.Base32hex)
	return sliceCid.Encode(encoder)
}

func uint64ToBytes(n uint64) []byte {
	byteBuf := bytes.NewBuffer([]byte{})
	binary.Write(byteBuf, binary.BigEndian, n)
	return byteBuf.Bytes()
}

func calcFileHash(data []byte) string {
	fileHash, _ := mh.Sum(data, mh.KECCAK_256, hashLen)
	fileCid := cid.NewCidV1(cid.Raw, fileHash)
	encoder, _ := mbase.NewEncoder(mbase.Base32hex)
	return fileCid.Encode(encoder)
}

func VerifyHash(hash string) bool {
	fileCid, err := cid.Decode(hash)
	if err != nil {
		return false
	}

	prefix := fileCid.Prefix()
	return prefix == hashCidPrefix
}
