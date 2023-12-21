package utils

import (
	"testing"
)

func TestIdWorker(t *testing.T) {
	idMap := make(map[int64]bool)

	idWorker, _ := NewIdWorker(int64(0))

	for i := 0; i < 1000000; i++ {
		id, _ := idWorker.NextId()
		if _, ok := idMap[id]; ok {
			t.Fatal("Found duplicate id")
		}
		idMap[id] = true
	}
}
