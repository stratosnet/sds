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

	"github.com/stratosnet/sds/framework/utils/crypto"
)

const (
	hashLen     = 20   // take 20 bytes of 32 bytes of hash
	VIDEO_CODEC = 0x72 // VIDEO_CODEC is separate from SDS_CODEC in order to identify the videos
	SDS_CODEC   = 0x66 // codec of legacy file hash is cid.RAW. New file hash uses SDS_CODEC.

	VALID_CID_VERSION = 1
	VALID_MH_TYPE     = 27
	VALID_MH_LENGTH   = 20
)

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

func CalcMD5OfSplitFiles(filePath []string) []byte {
	MD5 := md5.New()
	for _, path := range filePath {
		file, err := os.Open(path)
		if err != nil {
			Log(err.Error())
			return nil
		}
		defer file.Close()
		_, _ = io.Copy(MD5, file)
	}
	return MD5.Sum(nil)
}

func CalcFileKeccak(filePath string) mh.Multihash {
	file, err := os.Open(filePath)
	if err != nil {
		Log(err.Error())
		return nil
	}
	defer file.Close()

	sliceKeccak256, _ := mh.SumStream(file, mh.KECCAK_256, hashLen)
	return sliceKeccak256
}

func CalcKeccakOfSplitFiles(files []string) []byte {
	readers := make([]io.Reader, len(files))
	for i, filename := range files {
		file, err := os.Open(filename)
		if err != nil {
			return nil
		}
		defer file.Close()
		readers[i] = file
	}
	mergedReader := io.MultiReader(readers...)
	sliceKeccak256, _ := mh.SumStream(mergedReader, mh.KECCAK_256, hashLen)
	return sliceKeccak256
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

func CalcFileHash(filePath, encryptionTag string, codec byte) string {
	if filePath == "" {
		Log(errors.New("CalcFileHash: missing file path"))
		return ""
	}
	var data []byte
	switch codec {
	case cid.Raw:
		data = append([]byte(encryptionTag), CalcFileMD5(filePath)...)
	case VIDEO_CODEC, SDS_CODEC:
		data = append([]byte(encryptionTag), CalcFileKeccak(filePath)...)
	default:
		return ""
	}
	filehash, _ := mh.Sum(data, mh.KECCAK_256, hashLen)
	fileCid := cid.NewCidV1(uint64(codec), filehash)
	encoder, _ := mbase.NewEncoder(mbase.Base32hex)
	return fileCid.Encode(encoder)
}

func CalcFileHashFromSlices(files []string, encryptionTag string) string {
	data := append([]byte(encryptionTag), CalcKeccakOfSplitFiles(files)...)
	filehash, _ := mh.Sum(data, mh.KECCAK_256, hashLen)
	fileCid := cid.NewCidV1(uint64(SDS_CODEC), filehash)
	encoder, _ := mbase.NewEncoder(mbase.Base32hex)
	return fileCid.Encode(encoder)
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

// ValidateHash only validate the hash format, does NOT verify if the hash is created by certain content
func ValidateHash(hash string) bool {
	fileCid, err := cid.Decode(hash)
	if err != nil {
		return false
	}
	prefix := fileCid.Prefix()

	return prefix.Version == VALID_CID_VERSION && prefix.MhType == VALID_MH_TYPE && prefix.MhLength == VALID_MH_LENGTH
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
