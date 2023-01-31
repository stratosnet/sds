package hashring

import (
	"strconv"
	"testing"
	"time"

	"github.com/bsipos/thist"
	"github.com/google/uuid"
	"github.com/stratosnet/sds/utils"
)

func init() {
	utils.NewDefaultLogger("", false, false)
}

func TestWHRTrend(t *testing.T) {
	testRing := NewWeightedHashRing()

	start := time.Now()
	numNode := 100
	for i := 1; i <= numNode; i++ {
		tmpVWN := &WeightedNode{ID: "stsdsp2p1faej5w4q6hgnt0ft598dlm408g4p747ymg5jq6_" + strconv.Itoa(i), Host: "127.0.0.1:18092_" + strconv.Itoa(i), Rest: "127.0.0.1:18092_" + strconv.Itoa(i), Tier: uint32(i)}
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
			ID:   id,
			Host: "",
			Rest: "",
			Tier: 1,
			Data: nil,
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
