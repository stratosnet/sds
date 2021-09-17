package hashring

type WeightedNode struct {
	ID            string
	OngoingWeight float64
}

type WeightedNodeList []WeightedNode

func (p WeightedNodeList) Len() int           { return len(p) }
func (p WeightedNodeList) Less(i, j int) bool { return p[i].OngoingWeight < p[j].OngoingWeight }
func (p WeightedNodeList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type WeightedHashRing struct {
	HashRing       *HashRing
	NodeTaskWeight map[string]float64 // map(NodeId => 17.00
}

func NewWeightedHashRing(numOfVNode uint32) *WeightedHashRing {
	hashRing := New(numOfVNode)
	nodeTaskWeight := make(map[string]float64)

	whr := new(WeightedHashRing)
	whr.HashRing = hashRing
	whr.NodeTaskWeight = nodeTaskWeight
	return whr
}
