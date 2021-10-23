package hashring

import (
	"github.com/bsipos/thist"
	"github.com/google/uuid"
	"github.com/stratosnet/sds/utils"
	"math"
	"strconv"
	"testing"
	"time"
)

func TestWHRTrend(t *testing.T) {
	testRing := NewWeightedHashRing()

	start := time.Now()
	numNode := 100
	for i := 1; i <= numNode; i++ {
		tmpVWN := &WeightedNode{ID: "stsdsp2p1faej5w4q6hgnt0ft598dlm408g4p747ymg5jq6_" + strconv.Itoa(i), Host: "127.0.0.1:18092_" + strconv.Itoa(i), Rest: "127.0.0.1:18092_" + strconv.Itoa(i), Weight: math.Exp(float64(i))}
		testRing.AddNode(tmpVWN)
		testRing.NodeStatus[tmpVWN.ID] = true
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
