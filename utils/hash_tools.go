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
const VIDEO_CODEC = 0x72

var hashCidPrefix = cid.Prefix{
	Version:  1,
	Codec:    85,
	MhType:   27,
	MhLength: 20,
}

var hashCidPrefixForVideoStream = cid.Prefix{
	Version:  1,
	Codec:    114,
	MhType:   27,
	MhLength: 20,
}

func CalcCRC32(data []byte) uint32 {
	iEEE := crc32.NewIEEE()
	_, _ = io.WriteString(iEEE, string(data))
	return iEEE.Sum32()
}

func CalcCRC32OfSlices(data [][]byte) uint32 {
	iEEE := crc32.NewIEEE()
	for _, d := range data {
		_, _ = io.WriteString(iEEE, string(d))
	}
	return iEEE.Sum32()
}

func CalcFileMD5(filePath string) []byte {
	file, err := os.Open(filePath)
	if err != nil {
		Log(err.Error())
		return nil
	}
	defer func() {
		_ = file.Close()
	}()
	MD5 := md5.New()
	_, _ = io.Copy(MD5, file)
	return MD5.Sum(nil)
}

func CalcFileCRC32(filePath string) uint32 {
	file, err := os.Open(filePath)
	if err != nil {
		Log(err.Error())
		return 0
	}
	defer func() {
		_ = file.Close()
	}()
	iEEE := crc32.NewIEEE()
	_, _ = io.Copy(iEEE, file)
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
	return CalcFileHashFromData(data, cid.Raw)
}

func CalcFileHashForVideoStream(filePath, encryptionTag string) string {
	if filePath == "" {
		Log(errors.New("CalcFileHash: missing file path"))
		return ""
	}
	data := append([]byte(encryptionTag), CalcFileMD5(filePath)...)
	return CalcFileHashFromData(data, VIDEO_CODEC)
}

func CalcHash(data []byte) string {
	return hex.EncodeToString(crypto.Keccak256(data))
}

func CalcHashBytes(data []byte) []byte {
	return crypto.Keccak256(data)
}

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
	_ = binary.Write(byteBuf, binary.BigEndian, n)
	return byteBuf.Bytes()
}

func CalcFileHashFromData(data []byte, codec uint64) string {
	fileHash, _ := mh.Sum(data, mh.KECCAK_256, hashLen)
	fileCid := cid.NewCidV1(codec, fileHash)
	encoder, _ := mbase.NewEncoder(mbase.Base32hex)
	return fileCid.Encode(encoder)
}

func VerifyHash(hash string) bool {
	fileCid, err := cid.Decode(hash)
	if err != nil {
		return false
	}

	prefix := fileCid.Prefix()
	return prefix == hashCidPrefix || prefix == hashCidPrefixForVideoStream
}

func IsVideoStream(hash string) bool {
	code, err := GetCodecFromFileHash(hash)
	if err != nil {
		return false
	}
	return code == VIDEO_CODEC
}

func GetCodecFromFileHash(hash string) (uint64, error) {
	fileCid, err := cid.Decode(hash)
	if err != nil {
		return 0, err
	}

	return fileCid.Prefix().Codec, nil
}
