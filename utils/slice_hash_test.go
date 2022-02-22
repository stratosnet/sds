package utils

import (
	"fmt"
	"testing"
	"time"
)

func TestSliceHash(t *testing.T) {
	fileHash := calcFileHash([]byte("fileData"))

	//prepare data start
	sliceCnt := 300000
	sliceContentSize := 10 * 1024
	sliceList := make([][]byte, 0)

	for i := 0; i < sliceCnt; i++ {
		sliceBytes := make([]byte, 0)
		for j := 0; j < sliceContentSize; j++ {
			sliceBytes = append(sliceBytes, []byte("a")...)
		}
		sliceList = append(sliceList, sliceBytes)
	}
	//prepare data end
	start := time.Now()
	sliceNumber := uint64(0)
	for _, sliceData := range sliceList {
		_ = CalcSliceHash(sliceData, fileHash, sliceNumber)
		sliceNumber++
	}
	elapsed := time.Since(start)
	fmt.Println("300000 slices hash calculation : " + elapsed.String())
}
