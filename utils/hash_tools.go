package utils

import (
	"encoding/hex"
	"errors"
	"github.com/ipfs/go-cid"
	mbase "github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"

	"github.com/stratosnet/sds/utils/crypto"
)

// CalcCRC32
func CalcCRC32(data []byte) uint32 {
	iEEE := crc32.NewIEEE()
	io.WriteString(iEEE, string(data))
	return iEEE.Sum32()
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
	return calcFileHash(getFileData(filePath))
}

// CalcHash
func CalcHash(data []byte) string {
	return hex.EncodeToString(crypto.Keccak256(data))
}

// CalcHash
func CalcSliceHash(data []byte, fileHash string) string {
	fileCid, _ := cid.Decode(fileHash)
	fileKeccak256 := fileCid.Hash()
	sliceKeccak256, _ := mh.Sum(data, mh.KECCAK_256, 20)
	if len(fileKeccak256) != len(sliceKeccak256) {
		Log(errors.New("length of fileKeccak256 and sliceKeccak256 doesn't match"))
		return ""
	}
	sliceHash := make([]byte, len(fileKeccak256))
	for i := 0; i < len(fileKeccak256); i++ {
		sliceHash[i] = fileKeccak256[i] ^ sliceKeccak256[i]
	}
	sliceHash, _ = mh.Sum(sliceHash, mh.KECCAK_256, 20)
	sliceCid := cid.NewCidV1(cid.Raw, sliceHash)
	encoder, _ := mbase.NewEncoder(mbase.Base32hex)
	return sliceCid.Encode(encoder)
}

func getFileData(filePath string) []byte {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil
	}
	return data
}

func calcFileHash(data []byte) string {
	fileHash, _ := mh.Sum(data, mh.KECCAK_256, 20)
	fileCid := cid.NewCidV1(cid.Raw, fileHash)
	encoder, _ := mbase.NewEncoder(mbase.Base32hex)
	return fileCid.Encode(encoder)
}
