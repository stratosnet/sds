package utils

import (
	"crypto/rand"
	"testing"

	"github.com/ipfs/go-cid"
	mbase "github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"

	"github.com/stratosnet/sds/framework/crypto"
)

func BenchmarkSliceHash(b *testing.B) {
	filehash, _ := mh.Sum([]byte("fileData"), mh.KECCAK_256, 20)
	fileCid := cid.NewCidV1(uint64(cid.Raw), filehash)
	encoder, _ := mbase.NewEncoder(mbase.Base32hex)
	fh := fileCid.Encode(encoder)

	//prepare data
	sliceCnt := 1000
	sliceContentSize := 10 * 1024
	var sliceList [][]byte
	for i := 0; i < sliceCnt; i++ {
		bytes := make([]byte, sliceContentSize)
		_, err := rand.Read(bytes)
		if err != nil {
			b.Fatal(err)
		}
		sliceList = append(sliceList, bytes)
	}

	b.Run("BenchmarkSliceHash", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = crypto.CalcSliceHash(sliceList[i%sliceCnt], fh, uint64(i))
		}
	})
}
