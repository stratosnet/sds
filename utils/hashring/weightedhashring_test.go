package hashring

import (
	"strconv"
	"testing"
	"time"

	"github.com/bsipos/thist"
	"github.com/google/uuid"
	"github.com/stratosnet/sds/utils"
	"github.com/stretchr/testify/assert"
)

func init() {
	utils.NewDefaultLogger("", false, false)
}

func TestWHRTrend(t *testing.T) {
	testRing := NewWeightedHashRing()

	start := time.Now()
	numNode := 100
	for i := 1; i <= numNode; i++ {
		tmpVWN := &WeightedNode{ID: "stsdsp2p1faej5w4q6hgnt0ft598dlm408g4p747ymg5jq6_" + strconv.Itoa(i), Host: "127.0.0.1:18092_" + strconv.Itoa(i), Rest: "127.0.0.1:18092_" + strconv.Itoa(i), Copies: uint32(i * i)}
		testRing.AddNode(tmpVWN)
		testRing.NodeStatus.Store(tmpVWN.ID, true)
	}
	logPointA := time.Now()
	utils.DebugLogf("it cost %v(s) for creating a weighted hashring of %v nodes with numOfCopies [1-%v]", logPointA.Sub(start).Seconds(), numNode, numNode)

	numDraw := 1000000

	startDraw := time.Now()
	drawnStat := make(map[string]int)
	for i := 0; i < numDraw; i++ {
		ranStr := uuid.New().String()
		_, drawnID := testRing.GetNode("36226c5c6249a67510a0af7ea4b2b0b4736b4cbb1372f23b2a904ca0dbf40ab9#" + ranStr)
		existingDrawnNum := drawnStat[drawnID]
		drawnStat[drawnID] = existingDrawnNum + 1
	}
	logPointB := time.Now()
	totalDrawDuration := logPointB.Sub(startDraw)
	utils.DebugLogf("it cost %v(s) for %v draws, average draw time is %v(s)", totalDrawDuration, numDraw, totalDrawDuration.Seconds()/float64(numDraw))

	h := thist.NewHist(nil, "\n       Draw Stat histogram\n", "auto", 100, true)
	c := make(chan float64, numNode)
	dataToDraw := make([]int, 0)
	for i := 1; i <= numNode; i++ {
		dataToDraw = append(dataToDraw, drawnStat["stsdsp2p1faej5w4q6hgnt0ft598dlm408g4p747ymg5jq6_"+strconv.Itoa(i)])
	}

	utils.DebugLogf("========= total number of draws: %v =========\n", numDraw)
	for i, v := range dataToDraw {
		utils.DebugLogf("node %v with %v copies is drawn for %v times", i+1, i+1, v)
		// draw histogram
		for j := 1; j <= v; j++ {
			c <- float64(i + 1)
			h.Update(<-c)
		}
	}
	utils.DebugLog(h.Draw())
}

func TestGetNodeExcludedNodeIDsWeightedHashring(t *testing.T) {
	ring := NewWeightedHashRing()

	for i := 0; i < 5; i++ {
		id := "ID#" + strconv.FormatInt(int64(i), 10)
		ring.AddNode(&WeightedNode{
			ID:     id,
			Host:   "",
			Rest:   "",
			Copies: 1,
			Data:   nil,
		})
		ring.SetOnline(id)
	}

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

func TestWeightedHashringNodeCount(t *testing.T) {
	ring := NewWeightedHashRing()
	assert.Equal(t, uint32(0), ring.NodeCount)
	assert.Equal(t, uint32(0), ring.NodeOkCount)

	ring.AddNode(&WeightedNode{ID: "node1"})
	ring.AddNode(&WeightedNode{ID: "node2"})
	ring.AddNode(&WeightedNode{ID: "node3"})
	assert.Equal(t, uint32(3), ring.NodeCount)
	assert.Equal(t, uint32(0), ring.NodeOkCount)

	ring.SetOnline("node1")
	ring.SetOnline("node2")
	ring.SetOnline("node3")
	assert.Equal(t, uint32(3), ring.NodeCount)
	assert.Equal(t, uint32(3), ring.NodeOkCount)

	ring.RemoveNode("node1")
	ring.SetOffline("node2")
	ring.SetOffline("node3")
	ring.RemoveNode("node3")
	assert.Equal(t, uint32(1), ring.NodeCount)
	assert.Equal(t, uint32(0), ring.NodeOkCount)
}

/*
BenchmarkWeightedRandomGetNodes/100_of_10000-16         	    3674	    291901 ns/op
BenchmarkWeightedRandomGetNodes/500_of_10000-16         	     776	   1465892 ns/op
BenchmarkWeightedRandomGetNodes/1000_of_10000-16        	     392	   3056415 ns/op
BenchmarkWeightedRandomGetNodes/5000_of_10000-16        	      40	  26610405 ns/op
BenchmarkWeightedRandomGetNodes/100_of_100000-16        	    3063	    391340 ns/op
BenchmarkWeightedRandomGetNodes/500_of_100000-16        	     582	   1968920 ns/op
BenchmarkWeightedRandomGetNodes/1000_of_100000-16       	     292	   4039905 ns/op
BenchmarkWeightedRandomGetNodes/5000_of_100000-16       	      60	  20707943 ns/op
*/
func BenchmarkWeightedRandomGetNodes(b *testing.B) {
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
		ring := createRandomWeightedHashring(t.hashringCount)
		b.Run(t.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ring.RandomGetNodes(t.nodeCount)
			}
		})
	}
}

func createRandomWeightedHashring(count int) *WeightedHashRing {
	ring := NewWeightedHashRing()

	for i := 0; i < count; i++ {
		id := "ID#" + strconv.FormatInt(int64(i), 10)
		ring.AddNode(&WeightedNode{
			ID:     id,
			Host:   "",
			Rest:   "",
			Copies: 1,
			Data:   nil,
		})
		ring.SetOnline(id)
	}
	return ring
}
