package hashring

// hashring for consistency
// two improvement
// 1. add virtual node: when delete a node, have better distribution of load to other node
// 2. add red-black tree structure, improving search node efficiency
//

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"

	"github.com/HuKeping/rbtree"
	"github.com/stratosnet/sds/utils"
)

// Node
type WeightedNode struct {
	ID   string
	Host string
	Rest string
	Tier uint32
	Data *sync.Map
}

// nodeKey
func (n *WeightedNode) nodeKey() string {
	return n.ID + "#" + n.Host
}

// Less of rbtree
//
func (n *WeightedNode) Less(than rbtree.Item) bool {
	return utils.CalcCRC32([]byte(n.ID)) < utils.CalcCRC32([]byte(than.(*WeightedNode).ID))
}

// VWeightedNode virtual node
type VWeightedNode struct {
	Index  uint32 // index, crc32 of hashkey
	NodeID string
}

// Less of rbtree
func (vwn *VWeightedNode) Less(than rbtree.Item) bool {
	return vwn.Index < than.(*VWeightedNode).Index
}

// HashRing
type WeightedHashRing struct {
	VRing           *rbtree.Rbtree
	NRing           *rbtree.Rbtree
	Nodes           *sync.Map // map(NodeID => *WeightedNode)
	NodeStatus      *sync.Map // map(NodeID => status)
	NodeCount       uint32
	NodeOkCount     uint32
	NumberOfVirtual uint32
	sync.Mutex
}

// virtualKey
func (r *WeightedHashRing) virtualKey(nodeID string, index uint32) string {
	return "node#" + nodeID + "#" + strconv.FormatUint(uint64(index), 10)
}

// hashKey
func (r *WeightedHashRing) hashKey(key string) string {
	return utils.CalcHash([]byte(key))
}

// hashTOCRC32
func (r *WeightedHashRing) hashToCRC32(hashInString string) uint32 {
	return utils.CalcCRC32([]byte(hashInString))
}

// CalcIndex
func (r *WeightedHashRing) CalcIndex(key string) uint32 {
	return r.hashToCRC32(r.hashKey(key))
}

// AddNode
func (r *WeightedHashRing) AddNode(node *WeightedNode) {

	r.Lock()

	defer r.Unlock()

	// calc numOfCopies with node tier, WeightedNode should have at least 1 copy
	numOfCopies := getNumOfCopies(node.Tier)
	//utils.DebugLogf("for node %v (Tier=%v), numOfCopies is %v", node.ID, node.Tier, numOfCopies)
	var i uint32
	for i = 0; i < uint32(numOfCopies); i++ {
		index := r.CalcIndex(r.virtualKey(node.ID, i))
		//utils.DebugLogf("---- index is %v", index)
		r.VRing.Insert(&VWeightedNode{Index: index, NodeID: node.ID})
	}

	r.Nodes.Store(node.ID, node)
	r.NodeStatus.Store(node.ID, false)

	r.NRing.Insert(node)

	r.NodeCount++
}

func getNumOfCopies(nodeTier uint32) float64 {
	// numOfCopies = nodeTier ^ 2
	return math.Round(math.Pow(float64(nodeTier), 2))
}

// RemoveNode
func (r *WeightedHashRing) RemoveNode(nodeID string) bool {
	r.Lock()
	defer r.Unlock()

	val, ok := r.Nodes.Load(nodeID)
	if !ok {
		return true
	}
	node := val.(*WeightedNode)

	var numberOfNode uint32 = 1
	if r.NumberOfVirtual > 0 {
		numberOfNode = r.NumberOfVirtual
	}

	var i uint32
	for i = 0; i < numberOfNode; i++ {
		index := r.CalcIndex(r.virtualKey(node.ID, i))
		r.VRing.Delete(&VWeightedNode{Index: index, NodeID: node.ID})
	}

	r.Nodes.Delete(node.ID)
	r.NodeStatus.Delete(node.ID)

	r.NRing.Delete(node)

	r.NodeCount--

	return true
}

func (r *WeightedHashRing) Node(ID string) *WeightedNode {
	if node, ok := r.Nodes.Load(ID); ok {
		return node.(*WeightedNode)
	}
	return nil
}

func (r *WeightedHashRing) IsOnline(ID string) bool {
	online, ok := r.NodeStatus.Load(ID)
	return ok && online.(bool)
}

func (r *WeightedHashRing) SetOffline(ID string) {
	r.Lock()
	defer r.Unlock()

	if online, ok := r.NodeStatus.Load(ID); ok && online.(bool) {
		r.NodeStatus.Store(ID, false)
		r.NodeOkCount--
	}
}

func (r *WeightedHashRing) SetOnline(ID string) {
	r.Lock()
	defer r.Unlock()

	if online, ok := r.NodeStatus.Load(ID); ok && !online.(bool) {
		r.NodeOkCount++
		r.NodeStatus.Store(ID, true)
	}
}

func (r *WeightedHashRing) RandomGetNodes(num int) []*WeightedNode {

	if r.NodeOkCount <= 0 {
		return nil
	}

	if r.NodeOkCount < uint32(num) {
		num = int(r.NodeOkCount)
	}

	nodes := make([]*WeightedNode, num)

	ids := make([]string, 0)
	r.NodeStatus.Range(func(key, value interface{}) bool {
		id := key.(string)
		ok := value.(bool)
		if ok {
			ids = append(ids, id)
		}
		return true
	})

	indexes := utils.GenerateRandomNumber(0, len(ids), num)

	for i, idx := range indexes {
		if node, ok := r.Nodes.Load(ids[idx]); ok {
			nodes[i] = node.(*WeightedNode)
		}
	}

	return nodes
}

// GetNode
// @params key
func (r *WeightedHashRing) GetNode(key string) (uint32, string) {
	keyIndex := r.CalcIndex(key)
	//utils.DebugLogf("calc key index is %v", keyIndex)
	return r.GetNodeByIndex(keyIndex)
}

// GetNodeMissNodeID get node excluded given NodeIDs
// @params key
func (r *WeightedHashRing) GetNodeExcludedNodeIDs(key string, NodeIDs []string) (uint32, string) {

	if len(NodeIDs) <= 0 {
		return r.GetNode(key)
	}

	if uint32(len(NodeIDs)) >= r.NodeCount || r.NodeCount <= 0 {
		return 0, ""
	}

	for _, id := range NodeIDs {
		r.SetOffline(id)
	}

	index, id := r.GetNode(key)
	return index, id

	//tmpRing := New(r.NumberOfVirtual)
	//r.Nodes.Range(func(key, value interface{}) bool {
	//	node := value.(*WeightedNode)
	//	if !utils.StrInSlices(NodeIDs, node.ID) {
	//		tmpRing.AddNode(&WeightedNode{
	//			ID:   node.ID,
	//			Host: node.Host,
	//		})
	//	}
	//	return true
	//})
	//
	//if tmpRing.NodeCount > 0 {
	//	return tmpRing.GetNode(key)
	//}

}

// GetNodeUpDownNodes get upstream of downstream of node
// @params
func (r *WeightedHashRing) GetNodeUpDownNodes(NodeID string) (string, string) {
	online, ok := r.NodeStatus.Load(NodeID)
	if NodeID == "" || !ok || !online.(bool) || r.NodeCount <= 0 {
		return "", ""
	}

	if r.NRing.Len() <= 1 {
		return "", ""
	}

	up := r.NRing.Max().(*WeightedNode).ID
	down := r.NRing.Min().(*WeightedNode).ID

	r.NRing.Descend(&WeightedNode{ID: NodeID}, func(item rbtree.Item) bool {
		if utils.CalcCRC32([]byte(NodeID)) == utils.CalcCRC32([]byte(item.(*WeightedNode).ID)) {
			return true
		}
		up = item.(*WeightedNode).ID
		return false
	})

	r.NRing.Ascend(&WeightedNode{ID: NodeID}, func(item rbtree.Item) bool {
		if utils.CalcCRC32([]byte(NodeID)) == utils.CalcCRC32([]byte(item.(*WeightedNode).ID)) {
			return true
		}
		down = item.(*WeightedNode).ID
		return false
	})

	return up, down
}

// GetNodeByIndex
// @params keyIndex
func (r *WeightedHashRing) GetNodeByIndex(keyIndex uint32) (uint32, string) {

	if r.VRing.Len() <= 0 {
		return 0, ""
	}

	minVWeightedNodeOfRing := r.VRing.Min().(*VWeightedNode)

	vWeightedNode := minVWeightedNodeOfRing

	r.VRing.Ascend(&VWeightedNode{Index: keyIndex}, func(item rbtree.Item) bool {
		vWeightedNode = item.(*VWeightedNode)
		if online, ok := r.NodeStatus.Load(vWeightedNode.NodeID); ok && !online.(bool) {
			return true
		}
		return false
	})

	if online, ok := r.NodeStatus.Load(vWeightedNode.NodeID); ok && !online.(bool) {
		r.VRing.Ascend(minVWeightedNodeOfRing, func(item rbtree.Item) bool {
			vWeightedNode = item.(*VWeightedNode)
			if online, ok := r.NodeStatus.Load(vWeightedNode.NodeID); ok && !online.(bool) {
				return true
			}
			return false
		})
	}

	return vWeightedNode.Index, vWeightedNode.NodeID
}

// PrintNodes print all non-virtual nodes
func (r *WeightedHashRing) PrintNodes() {

	if r.NodeCount <= 0 {
		fmt.Println("nodes is empty")
		return
	}

	r.Nodes.Range(func(key, value interface{}) bool {
		node := value.(*WeightedNode)
		fmt.Println(strings.Repeat("=", 30))
		fmt.Println("NodeID:", node.ID)
		fmt.Println("NodeHost:", node.Host)
		fmt.Println("NodeKey :", node.nodeKey())
		fmt.Println()
		return true
	})
}

// TraversalVRing traverse virtual rbtree
func (r *WeightedHashRing) TraversalVRing() {
	r.VRing.Ascend(r.VRing.Min(), func(item rbtree.Item) bool {
		fmt.Printf("VWeightedNode %d => %s\n", item.(*VWeightedNode).Index, item.(*VWeightedNode).NodeID)
		return true
	})
}

// TraversalNRing traverse non-virtual rbtree
func (r *WeightedHashRing) TraversalNRing() {
	r.NRing.Ascend(r.NRing.Min(), func(item rbtree.Item) bool {
		fmt.Printf("Node %d => %s\n", utils.CalcCRC32([]byte(item.(*WeightedNode).ID)), item.(*WeightedNode).ID)
		return true
	})
}

// NewHashRing
func NewWeightedHashRing() *WeightedHashRing {
	r := new(WeightedHashRing)
	r.Nodes = new(sync.Map)
	r.NodeStatus = new(sync.Map)
	r.NodeCount = 0
	//r.NumberOfVirtual = numOfVNode

	r.VRing = rbtree.New()
	r.NRing = rbtree.New()
	return r
}
