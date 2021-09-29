package hashring

import (
	"errors"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/database"
	"math/rand"
	"sync"
)

type AccumulatedTaskWeight []float64

type WeightedRing interface {
	WeightDrawExcludeNodeIDs(int, []string) ([]interface{}, error)
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

	mutex sync.Mutex
}

func (whr *WeightedHashRing) AddNode(node *Node) {
	utils.DebugLogf("adding node [%v] to hashring", node.ID)
	whr.hashRing.AddNode(node)
	// test only - to be removed later
	//whr.hashRing.SetOnline(node.ID)
	//if whr.hashRing.NodeCount >= 3 {
	//	_, err := whr.WeightDrawExcludeNodeIDs(10, nil)
	//	utils.DebugLogf("nodeTaskWeight to-date is %v, %v", whr.nodeTaskWeight, err)
	//}
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

func (whr *WeightedHashRing) WeightDrawExcludeNodeIDs(rollTimes int, nodeIDsToExclude []string) ([]interface{}, error) {
	whr.mutex.Lock()
	defer whr.mutex.Unlock()
	retNodeIDs := make([]interface{}, 0)
	maxAcmWeights := whr.refreshIndexByWeight(nodeIDsToExclude)
	if maxAcmWeights == float64(0) {
		return nil, errors.New("no available node in hashring")
	}
	success := 0
	for success < rollTimes {
		// get a rand acmWeight, min = 0, max = maxAcmWeights
		randAcmWeight := rand.Float64() * maxAcmWeights
		// check node index according to randAcmWeight
		candidateIndex := 0
		for i, val := range whr.acmWeight {
			if randAcmWeight > val {
				continue
			}
			candidateIndex = i
			break
		}
		retNodeIDs = append(retNodeIDs, whr.nodeIndex[candidateIndex])
		utils.DebugLogf("draw no.%v is %v", success, retNodeIDs[success])
		success++
	}
	utils.DebugLogf("all draws are %v", retNodeIDs)
	return retNodeIDs, nil
}

func (whr *WeightedHashRing) refreshIndexByWeight(nodeIDsToExclude []string) float64 {
	// reload latest weights for VNodes
	whr.nodeTaskWeight = whr.reloadWeightsForVNodes()
	// exclude node IDs
	tmpNodeTaskWeight := whr.nodeTaskWeight
	for _, k := range nodeIDsToExclude {
		delete(tmpNodeTaskWeight, k)
	}
	// exclude offline nodes
	var filteredNodeTaskWeight = make(map[string]float64)
	for k, v := range tmpNodeTaskWeight {
		if whr.hashRing.IsOnline(k) {
			filteredNodeTaskWeight[k] = v
		}
		utils.DebugLogf("whr.hashRing.NodeStatus[%s] is %v", k, whr.hashRing.NodeStatus[k])
	}
	if len(filteredNodeTaskWeight) < 1 {
		return float64(0)
	}
	utils.DebugLogf("reloaded nodeTaskWeight are %v, after exclusion are %v", whr.nodeTaskWeight, filteredNodeTaskWeight)
	// update acmWeight and nodeIndex by nodeTaskWeight
	var tmpAcmWeight AccumulatedTaskWeight
	tmpNodeIndex := make(map[int]string)
	i := 0
	for k, v := range filteredNodeTaskWeight {
		if len(tmpNodeIndex) == 0 {
			tmpAcmWeight = append(tmpAcmWeight, v)
			tmpNodeIndex[0] = k
			i++
			continue
		}
		tmpAcmWeight = append(tmpAcmWeight, v+tmpAcmWeight[i-1])
		tmpNodeIndex[i] = k
		i++
	}
	utils.DebugLogf("updated acmWeight and nodeIndex are %v and %v", tmpAcmWeight, tmpNodeIndex)
	whr.acmWeight = tmpAcmWeight
	whr.nodeIndex = tmpNodeIndex
	// return last acmWeight, for further draws
	return tmpAcmWeight[len(tmpAcmWeight)-1]
}

func NewWeightedHashRing(numOfVNode uint32) *WeightedHashRing {
	hashRing := New(numOfVNode)
	nodeTaskWeight := make(map[string]float64)
	nodeIndex := make(map[int]string)
	var acmTaskWeight = make([]float64, 0, numOfVNode)

	whr := new(WeightedHashRing)
	whr.hashRing = hashRing
	whr.nodeTaskWeight = nodeTaskWeight
	whr.nodeIndex = nodeIndex
	whr.acmWeight = acmTaskWeight
	return whr
}
