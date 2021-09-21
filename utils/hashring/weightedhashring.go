package hashring

import (
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/database"
	"sync"
)

type AccumulatedTaskWeight []float64

type WeightedRing interface {
	//ReloadWeightsForOnlineNodes() map[string]float64
	//RefreshIndexByWeight()
	GetNodeIDByIndex(index int) string
}

type WeightQuerier func(*database.CacheTable, []interface{}) map[string]float64

type WeightedHashRing struct {
	WeightedRing
	hashRing       *HashRing
	nodeTaskWeight map[string]float64    // map(NodeId => 17.00
	acmWeight      AccumulatedTaskWeight // [100.00, 400.32, 600.88, 742.30]
	nodeIndex      map[int]string        // 0 -> nodeID3, 1 -> nodeID2, 2 -> nodeID5
	ct             *database.CacheTable
	querier        WeightQuerier

	mu sync.Mutex
}

func (whr *WeightedHashRing) AddNode(node *Node) {
	utils.DebugLogf("adding node [%v] to hashring", node.ID)
	whr.hashRing.AddNode(node)
}

func (whr *WeightedHashRing) RemoveNode(nodeID string) {
	whr.hashRing.RemoveNode(nodeID)
	//delete(whr.nodeTaskWeight, nodeID)
}

func (whr *WeightedHashRing) SetOnline(nodeID string) {
	whr.hashRing.SetOnline(nodeID)
}

func (whr *WeightedHashRing) SetOffline(nodeID string) {
	whr.hashRing.SetOffline(nodeID)
	//delete(whr.nodeTaskWeight, nodeID)
}

func (whr *WeightedHashRing) GetHashRing() *HashRing {
	return whr.hashRing
}

func (whr *WeightedHashRing) GetNodeTaskWeight() map[string]float64 {
	return whr.nodeTaskWeight
}

func (whr *WeightedHashRing) GetAcmWeight() AccumulatedTaskWeight {
	return whr.acmWeight
}

func (whr *WeightedHashRing) WithCacheTable(ct *database.CacheTable) *WeightedHashRing {
	whr.ct = ct
	return whr
}

func (whr *WeightedHashRing) WithWeightQuerier(querier func(*database.CacheTable, []interface{}) map[string]float64) *WeightedHashRing {
	whr.querier = querier
	return whr
}

func (whr *WeightedHashRing) reloadWeightsForVNodes() map[string]float64 {
	if whr.querier == nil || whr.ct == nil {
		utils.DebugLog("no weight querier or cTable specified")
		var ret = make(map[string]float64, 0)
		return ret
	}
	if whr.hashRing.NodeCount <= 0 {
		var ret = make(map[string]float64, 0)
		return ret
	}
	var nodeIDs = make([]interface{}, 0, whr.hashRing.NodeCount)
	whr.hashRing.Nodes.Range(func(key, value interface{}) bool {
		nodeID := value.(*Node).ID
		nodeIDs = append(nodeIDs, nodeID)
		return true
	})
	return whr.querier(whr.ct, nodeIDs)
}

func (whr *WeightedHashRing) refreshIndexByWeight() {
	whr.nodeTaskWeight = whr.reloadWeightsForVNodes()
}

func NewWeightedHashRing(numOfVNode uint32) *WeightedHashRing {
	hashRing := New(numOfVNode)
	nodeTaskWeight := make(map[string]float64)
	nodeIndex := make(map[int]string)
	var acmTaskWeight = make([]float64, numOfVNode)

	whr := new(WeightedHashRing)
	whr.hashRing = hashRing
	whr.nodeTaskWeight = nodeTaskWeight
	whr.nodeIndex = nodeIndex
	whr.acmWeight = acmTaskWeight
	return whr
}

type WeightedNode struct {
	ID            string
	OngoingWeight float64
}

type WeightedNodeList []WeightedNode

func (p WeightedNodeList) Len() int           { return len(p) }
func (p WeightedNodeList) Less(i, j int) bool { return p[i].OngoingWeight < p[j].OngoingWeight }
func (p WeightedNodeList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
