package hashring

type WeightedNode struct {
	ID            string
	OngoingWeight float64
}

type WeightedNodeList []WeightedNode

func (p WeightedNodeList) Len() int           { return len(p) }
func (p WeightedNodeList) Less(i, j int) bool { return p[i].OngoingWeight < p[j].OngoingWeight }
func (p WeightedNodeList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type AccumulatedTaskWeight []float64

type WeightedHashRing struct {
	HashRing       *HashRing
	NodeTaskWeight map[string]float64 // map(NodeId => 17.00
	AcmWeight      AccumulatedTaskWeight
}

func (whr *WeightedHashRing) AddWeightedNode(node *Node) {
	whr.HashRing.AddNode(node)
	// TODO get weight
	whr.NodeTaskWeight[node.ID] = 0.00
}

func (whr *WeightedHashRing) RemoveWeightedNode(nodeID string) {
	whr.HashRing.RemoveNode(nodeID)
	delete(whr.NodeTaskWeight, nodeID)
}

func (whr *WeightedHashRing) SetOnlineWeighted(nodeID string) {
	whr.HashRing.SetOnline(nodeID)
}

func (whr *WeightedHashRing) SetOfflineWeighted(nodeID string) {
	whr.HashRing.SetOffline(nodeID)
}

func (whr *WeightedHashRing) RefreshAcmWeights() {

}

func NewWeightedHashRing(numOfVNode uint32) *WeightedHashRing {
	hashRing := New(numOfVNode)
	nodeTaskWeight := make(map[string]float64)

	whr := new(WeightedHashRing)
	whr.HashRing = hashRing
	whr.NodeTaskWeight = nodeTaskWeight
	return whr
}
