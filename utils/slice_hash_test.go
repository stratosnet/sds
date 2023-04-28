package utils

import (
	"crypto/rand"
	"testing"

	"github.com/ipfs/go-cid"
)

func BenchmarkSliceHash(b *testing.B) {
	fileHash := CalcFileHashFromData([]byte("fileData"), cid.Raw)

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
			_ = CalcSliceHash(sliceList[i%sliceCnt], fileHash, uint64(i))
		}
	})
}
