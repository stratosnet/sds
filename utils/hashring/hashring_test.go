package hashring

import (
	"strconv"
	"testing"
)

func TestGetNodeExcludedNodeIDs(t *testing.T) {
	ring := createRandomHashring(5)

	if ring.NodeOkCount != 5 {
		t.Fatalf("Wrong NodeOkCount [%v] (expected [%v])", ring.NodeOkCount, 5)
	}

	_, nodeId := ring.GetNodeExcludedNodeIDs("some key", []string{"ID#1", "ID#2", "ID#3", "ID#4"}, false)
	expectedId := "ID#0"
	if nodeId != expectedId {
		t.Fatalf("Wrong node ID found [%v] (expected [%v])", nodeId, expectedId)
	}

	if ring.NodeOkCount != 5 {
		t.Fatalf("Wrong NodeOkCount [%v] (expected [%v])", ring.NodeOkCount, 5)
	}

	_, nodeId = ring.GetNodeExcludedNodeIDs("some other key", []string{"ID#1", "ID#2", "ID#3", "ID#4"}, true)
	if nodeId != expectedId {
		t.Fatalf("Wrong node ID found [%v] (expected [%v])", nodeId, expectedId)
	}

	if ring.NodeOkCount != 1 {
		t.Fatalf("Wrong NodeOkCount [%v] (expected [%v])", ring.NodeOkCount, 1)
	}
}

/*
BenchmarkRandomGetNodes/100_of_10000-16         	    1980	    559972 ns/op
BenchmarkRandomGetNodes/500_of_10000-16         	    1807	    655991 ns/op
BenchmarkRandomGetNodes/1000_of_10000-16        	    1564	    739677 ns/op
BenchmarkRandomGetNodes/5000_of_10000-16        	     831	   1362630 ns/op
BenchmarkRandomGetNodes/100_of_100000-16        	      91	  12908153 ns/op
BenchmarkRandomGetNodes/500_of_100000-16        	     100	  12318512 ns/op
BenchmarkRandomGetNodes/1000_of_100000-16       	     100	  12000401 ns/op
BenchmarkRandomGetNodes/5000_of_100000-16       	     100	  13949903 ns/op
*/
func BenchmarkRandomGetNodes(b *testing.B) {
	tests := []struct {
		name          string
		hashringCount int
		nodeCount     int
	}{
		{"100 of 10000", 10000, 100},
		{"500 of 10000", 10000, 500},
		{"1000 of 10000", 10000, 1000},
		{"5000 of 10000", 10000, 5000},
		{"100 of 100000", 100000, 100},
		{"500 of 100000", 100000, 500},
		{"1000 of 100000", 100000, 1000},
		{"5000 of 100000", 100000, 5000},
	}
	for _, t := range tests {
		ring := createRandomHashring(t.hashringCount)
		b.Run(t.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ring.RandomGetNodes(t.nodeCount)
			}
		})
	}
}

func createRandomHashring(count int) *HashRing {
	ring := New(10)

	for i := 0; i < count; i++ {
		id := "ID#" + strconv.FormatInt(int64(i), 10)
		ring.AddNode(&Node{
			ID:   id,
			Host: "",
			Rest: "",
			Data: nil,
		})
		ring.SetOnline(id)
	}
	return ring
}
