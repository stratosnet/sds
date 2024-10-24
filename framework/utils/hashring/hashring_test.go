package hashring

import (
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/botond-sipos/thist"
	"github.com/google/uuid"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stretchr/testify/require"
)

func init() {
	utils.NewDefaultLogger("", true, false)
}

func TestGetNodeExcludedNodeIDs(t *testing.T) {
	ring := createRandomHashring(5)
	require.Equal(t, uint32(5), ring.NodeOkCount())

	for i := 0; i < 1000; i++ {
		nodeId, _ := ring.RandomNodeExcludedIDs([]string{"id#1", "id#2", "id#3", "id#4"}, "")
		require.Equal(t, "id#0", nodeId)
		require.Equal(t, uint32(5), ring.NodeOkCount())
	}
	nodeId, node := ring.RandomNodeExcludedIDs([]string{"id#0", "id#1", "id#2", "id#3", "id#4"}, "")
	require.Equal(t, nodeId, "")
	require.Nil(t, node)
}

func TestHashringNodeCount(t *testing.T) {
	ring := New()
	require.Equal(t, uint32(0), ring.NodeCount())
	require.Equal(t, uint32(0), ring.NodeOkCount())

	ring.AddNode("node1", &Node{})
	ring.AddNode("node2", &Node{})
	ring.AddNode("node3", &Node{})
	require.Equal(t, uint32(3), ring.NodeCount())
	require.Equal(t, uint32(0), ring.NodeOkCount())

	ring.SetOnline("node1")
	ring.SetOnline("node2")
	ring.SetOnline("node3")
	require.Equal(t, uint32(3), ring.NodeCount())
	require.Equal(t, uint32(3), ring.NodeOkCount())

	ring.RemoveNode("node1")
	ring.SetOffline("node2")
	ring.SetOffline("node3")
	ring.RemoveNode("node3")
	require.Equal(t, uint32(1), ring.NodeCount())
	require.Equal(t, uint32(0), ring.NodeOkCount())
}

func TestHashringRemoveNode(t *testing.T) {
	ring := New()
	ring.AddNode("node1", &Node{})
	ring.RemoveNode("node1")
	require.Equal(t, uint32(0), ring.NodeCount())

	ring.AddNode("node1", &Node{})
	ring.SetOnline("node1")
	ring.RemoveNode("node1")
	require.Equal(t, uint32(0), ring.NodeCount())
	require.Equal(t, uint32(0), ring.NodeOkCount())

	ring.AddNode("node1", &Node{})
	ring.AddNode("node2", &Node{})
	ring.SetOnline("node1")
	ring.SetOnline("node2")
	ring.RemoveNode("node1")
	require.Equal(t, uint32(1), ring.NodeCount())
	require.Equal(t, uint32(1), ring.NodeOkCount())
	ring.RemoveNode("node2")
	require.Equal(t, uint32(0), ring.NodeCount())
	require.Equal(t, uint32(0), ring.NodeOkCount())

	ring.AddNode("node1", &Node{})
	ring.AddNode("node2", &Node{})
	ring.SetOnline("node1")
	ring.SetOnline("node2")
	ring.RemoveNode("node2")
	require.Equal(t, uint32(1), ring.NodeCount())
	require.Equal(t, uint32(1), ring.NodeOkCount())
	ring.RemoveNode("node1")
	require.Equal(t, uint32(0), ring.NodeCount())
	require.Equal(t, uint32(0), ring.NodeOkCount())
}

func TestRandomTrends(t *testing.T) {
	baseId := "stsdsp2p1faej5w4q6hgnt0ft598dlm408g4p747ymg5jq6"
	numNode := 10
	tolerance := 0.02

	utils.DebugLog("Testing RandomNode trend for non-weighted hashring")
	numDraw := 500000
	ring := createHashringForTrends(numNode, false, baseId)
	drawnStat := calcTrend(ring, numDraw, 1, baseId)
	for i := 1; i <= numNode; i++ {
		id := fmt.Sprintf("%v_%v", baseId, i)
		expected := float64(numDraw) / float64(numNode)
		deviation := float64(drawnStat[id])/expected - 1
		require.LessOrEqualf(t, math.Abs(deviation), tolerance, "Deviation (%v) larger than tolerance (%v). Expected %v draws, got %v instead (non-weighted RandomNode)", deviation, tolerance, int(expected), drawnStat[id])
	}

	utils.DebugLog("Testing RandomNodes trend for non-weighted hashring")
	numDraw = 100000
	drawnStat = calcTrend(ring, numDraw, 5, baseId)
	for i := 1; i <= numNode; i++ {
		id := fmt.Sprintf("%v_%v", baseId, i)
		expected := 5 * float64(numDraw) / float64(numNode)
		deviation := float64(drawnStat[id])/expected - 1
		require.LessOrEqualf(t, math.Abs(deviation), tolerance, "Deviation (%v) larger than tolerance (%v). Expected %v draws, got %v instead (non-weighted RandomNodes)", deviation, tolerance, int(expected), drawnStat[id])
	}

	utils.DebugLog("Testing RandomNode trend for weighted hashring")
	numDraw = 1000000
	ring = createHashringForTrends(numNode, true, baseId)
	drawnStat = calcTrend(ring, numDraw, 1, baseId)
	sumOfWeights := float64(numNode * (numNode + 1) / 2)
	for i := 1; i <= numNode; i++ {
		id := fmt.Sprintf("%v_%v", baseId, i)
		weightRatio := float64(i) / sumOfWeights
		expected := weightRatio * float64(numDraw)
		deviation := float64(drawnStat[id])/expected - 1
		require.LessOrEqualf(t, math.Abs(deviation), tolerance, "Deviation (%v) larger than tolerance (%v). Expected %v draws, got %v instead (weighted RandomNode)", deviation, tolerance, int(expected), drawnStat[id])
	}

	utils.DebugLog("Testing RandomNodes trend for weighted hashring")
	drawnStat = calcTrend(ring, numDraw, 5, baseId)
	for i := 1; i <= numNode; i++ {
		id := fmt.Sprintf("%v_%v", baseId, i)
		weightRatio := float64(i) / sumOfWeights
		expected := 5 * weightRatio * float64(numDraw)
		deviation := float64(drawnStat[id])/expected - 1
		if math.Abs(deviation) > tolerance {
			utils.DebugLogf("Deviation (%v) larger than tolerance (%v). Expected %v draws, got %v instead (weighted RandomNodes)", deviation, tolerance, int(expected), drawnStat[id])
		}
	}
	utils.DebugLogf("You can ignore these deviations. Weighted random sampling without replacement doesn't have \n" +
		"an easy way to calculate the expected value, because the weight distribution changes with every selection.\n" +
		"As the number of samples increases, the lower weights become more likely to be chosen, and the higher weights less likely.")
}

/*
BenchmarkRandomGetNodes/100_of_1000-16         	       16203	     71308 ns/op
BenchmarkRandomGetNodes/500_of_1000-16         	       17024	     69569 ns/op
BenchmarkRandomGetNodes/1000_of_1000-16        	       16860	     70024 ns/op
BenchmarkRandomGetNodes/100_of_10000-16         	    1374	    803459 ns/op
BenchmarkRandomGetNodes/500_of_10000-16         	     411	   2880824 ns/op
BenchmarkRandomGetNodes/1000_of_10000-16        	     224	   5254755 ns/op
BenchmarkRandomGetNodes/5000_of_10000-16        	      55	  20349606 ns/op
BenchmarkRandomGetNodes/100_of_100000-16        	     151	   8680566 ns/op
BenchmarkRandomGetNodes/500_of_100000-16        	      40	  27771684 ns/op
BenchmarkRandomGetNodes/1000_of_100000-16       	      21	  54545008 ns/op
BenchmarkRandomGetNodes/5000_of_100000-16       	       4	 250368172 ns/op
*/
func BenchmarkRandomGetNodes(b *testing.B) {
	tests := []struct {
		name          string
		hashringCount int
		nodeCount     int
	}{
		{"100 of 1000", 1000, 100},
		{"500 of 1000", 1000, 100},
		{"1000 of 1000", 1000, 100},
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
				ring.RandomNodes(t.nodeCount, "")
			}
		})
	}
}

func createRandomHashring(count int) *HashRing {
	ring := New()

	for i := 0; i < count; i++ {
		id := "id#" + strconv.FormatInt(int64(i), 10)
		ring.AddNode(id, &Node{
			id:   id,
			Host: "",
			Rest: "",
			Data: nil,
		})
		ring.SetOnline(id)
	}
	return ring
}

func createHashringForTrends(count int, weighted bool, baseId string) *HashRing {
	ring := New()

	for i := 1; i <= count; i++ {
		node := &Node{Host: "127.0.0.1:18092_" + strconv.Itoa(i), Rest: "127.0.0.1:18092_" + strconv.Itoa(i)}
		if weighted {
			node.Weight = float64(i)
		}
		id := baseId + "_" + strconv.Itoa(i)
		ring.AddNode(id, node)
		ring.SetOnline(id)
	}
	return ring
}

func calcTrend(ring *HashRing, numDraw, nodesPerDraw int, baseId string) map[string]int {
	numNode := int(ring.NodeOkCount())

	startDraw := time.Now()
	drawnStat := make(map[string]int)
	for i := 0; i < numDraw; i++ {
		seed := "36226c5c6249a67510a0af7ea4b2b0b4736b4cbb1372f23b2a904ca0dbf40ab9#" + uuid.New().String()
		if nodesPerDraw == 1 {
			drawnID, _ := ring.RandomNode(seed)
			drawnStat[drawnID] += 1
		} else {
			nodes := ring.RandomNodes(nodesPerDraw, seed)
			for _, node := range nodes {
				drawnStat[node.id] += 1
			}
		}
	}
	logPointB := time.Now()
	totalDrawDuration := logPointB.Sub(startDraw)
	utils.DebugLogf("it cost %v(s) for %v draws, average draw time is %v(s)", totalDrawDuration, numDraw, totalDrawDuration.Seconds()/float64(numDraw))

	h := thist.NewHist(nil, "\n       Draw Stat histogram", "auto", 100, true)
	c := make(chan float64, numNode)
	dataToDraw := make([]int, 0)
	for i := 1; i <= numNode; i++ {
		dataToDraw = append(dataToDraw, drawnStat[fmt.Sprintf("%v_%v", baseId, i)])
	}

	utils.DebugLogf("========= total number of draws: %v =========", numDraw)
	for i, v := range dataToDraw {
		utils.DebugLogf("node %v is drawn %v times", i+1, v)
		// draw histogram
		for j := 1; j <= v; j++ {
			c <- float64(i + 1)
			h.Update(<-c)
		}
	}
	utils.DebugLog(h.Draw())

	return drawnStat
}
