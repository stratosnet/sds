package hashring

// hashring for consistency
// two improvement
// 1. add virtual node: when delete a node, have better distribution of load to other node
// 2. add red-black tree structure, improving search node efficiency
//

import (
	"fmt"
	"strings"
	"sync"

	"github.com/HuKeping/rbtree"
	"github.com/google/uuid"
	"github.com/stratosnet/sds/framework/crypto"
)

type WeightedNode struct {
	ID     string
	Host   string
	Rest   string
	Copies uint32 // Number of copies in the hashring
	Data   *sync.Map
}

func (n *WeightedNode) nodeKey() string {
	return n.ID + "#" + n.Host
}

// Less of rbtree
func (n *WeightedNode) Less(than rbtree.Item) bool {
	return crypto.CalcCRC32([]byte(n.ID)) < crypto.CalcCRC32([]byte(than.(*WeightedNode).ID))
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

func (r *WeightedHashRing) AddNode(node *WeightedNode) {
	r.Lock()
	defer r.Unlock()

	for i := uint32(0); i < node.Copies; i++ {
		index := calcIndex(virtualKey(node.ID, i))
		//utils.DebugLogf("---- index is %v", index)
		r.VRing.Insert(&VWeightedNode{Index: index, NodeID: node.ID})
	}

	r.Nodes.Store(node.ID, node)
	r.NodeStatus.Store(node.ID, false)

	r.NRing.Insert(node)

	r.NodeCount++
}

func (r *WeightedHashRing) RemoveNode(nodeID string) bool {
	r.Lock()
	defer r.Unlock()

	val, ok := r.Nodes.Load(nodeID)
	if !ok {
		return true
	}
	node := val.(*WeightedNode)

	if r.IsOnline(nodeID) {
		r.NodeOkCount--
	}

	for i := uint32(0); i < node.Copies; i++ {
		index := calcIndex(virtualKey(node.ID, i))
		r.VRing.Delete(&VWeightedNode{Index: index, NodeID: node.ID})
	}

	r.Nodes.Delete(node.ID)
	r.NodeStatus.Delete(node.ID)

	r.NRing.Delete(node)

	r.NodeCount--

	return true
}

func (r *WeightedHashRing) UpdateCopies(nodeID string, updatedCopies uint32) {
	node := r.Node(nodeID)
	if node == nil || node.Copies == updatedCopies {
		return
	}

	// Add missing copies
	for i := node.Copies; i < updatedCopies; i++ {
		index := calcIndex(virtualKey(nodeID, i))
		r.VRing.Insert(&VWeightedNode{Index: index, NodeID: nodeID})
	}

	// Remove surplus copies
	for i := updatedCopies; i < node.Copies; i++ {
		index := calcIndex(virtualKey(nodeID, i))
		r.VRing.Delete(&VWeightedNode{Index: index, NodeID: nodeID})
	}

	node.Copies = updatedCopies
	r.Nodes.Store(nodeID, node)
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

// RandomGetNodes return random nodes from the hashring
func (r *WeightedHashRing) RandomGetNodes(num int) []*WeightedNode {
	if r.NodeOkCount <= 0 {
		return nil
	}

	if r.NodeOkCount < uint32(num) {
		// Return all online nodes
		var nodes []*WeightedNode
		r.Nodes.Range(func(key, value interface{}) bool {
			id := key.(string)
			node := value.(*WeightedNode)
			if r.IsOnline(id) {
				nodes = append(nodes, node)
			}
			return true
		})
		return nodes
	}

	nodes := make([]*WeightedNode, num)
	taken := make(map[string]bool)
	for i := 0; i < num; i++ {
		_, nodeID := r.GetNode(uuid.New().String())
		if taken[nodeID] {
			i--
			continue
		}
		taken[nodeID] = true
		if !r.IsOnline(nodeID) {
			i--
			continue
		}
		nodes[i] = r.Node(nodeID)
	}
	return nodes
}

// GetNode calculates an index from the given key, and returns a node selected using this index
func (r *WeightedHashRing) GetNode(key string) (uint32, string) {
	keyIndex := calcIndex(key)
	//utils.DebugLogf("calc key index is %v", keyIndex)
	return r.GetNodeByIndex(keyIndex)
}

// GetNodeExcludedNodeIDs calculates an index from the given key, and returns a node selected using this index.
// The nodes with IDs specified by NodeIDs will be excluded. If setOffline is true, the excluded nodes will become offline.
func (r *WeightedHashRing) GetNodeExcludedNodeIDs(key string, NodeIDs []string, setOffline bool) (uint32, string) {
	if len(NodeIDs) <= 0 {
		return r.GetNode(key)
	}

	if uint32(len(NodeIDs)) >= r.NodeCount || r.NodeCount <= 0 {
		return 0, ""
	}

	var temporaryOffline []string
	for _, id := range NodeIDs {
		if r.IsOnline(id) {
			temporaryOffline = append(temporaryOffline, id)
			r.SetOffline(id)
		}
	}

	index, id := r.GetNode(key)

	if !setOffline {
		for _, offlineId := range temporaryOffline {
			r.SetOnline(offlineId)
		}
	}
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
		if crypto.CalcCRC32([]byte(NodeID)) == crypto.CalcCRC32([]byte(item.(*WeightedNode).ID)) {
			return true
		}
		up = item.(*WeightedNode).ID
		return false
	})

	r.NRing.Ascend(&WeightedNode{ID: NodeID}, func(item rbtree.Item) bool {
		if crypto.CalcCRC32([]byte(NodeID)) == crypto.CalcCRC32([]byte(item.(*WeightedNode).ID)) {
			return true
		}
		down = item.(*WeightedNode).ID
		return false
	})

	return up, down
}

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
		fmt.Printf("Node %d => %s\n", crypto.CalcCRC32([]byte(item.(*WeightedNode).ID)), item.(*WeightedNode).ID)
		return true
	})
}

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
