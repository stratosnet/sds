package hashring

// hashring for consistency
// two improvement
// 1. add virtual node: when delete a node, have better distribution of load to other node
// 2. add red-black tree structure, improving search node efficiency
//

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/HuKeping/rbtree"
	"github.com/stratosnet/sds/utils"
)

// Node
type Node struct {
	ID   string
	Host string
	Rest string
	Data *sync.Map
}

// nodeKey
func (n *Node) nodeKey() string {
	return n.ID + "#" + n.Host
}

// Less of rbtree
//
func (n *Node) Less(than rbtree.Item) bool {
	return utils.CalcCRC32([]byte(n.ID)) < utils.CalcCRC32([]byte(than.(*Node).ID))
}

// VNode virtual node
type VNode struct {
	Index  uint32 // index, crc32 of hashkey
	NodeID string
}

// Less of rbtree
func (vn *VNode) Less(than rbtree.Item) bool {
	return vn.Index < than.(*VNode).Index
}

// HashRing
type HashRing struct {
	VRing           *rbtree.Rbtree
	NRing           *rbtree.Rbtree
	Nodes           *sync.Map // map(NodeID => *Node)
	NodeStatus      *sync.Map // map(NodeID => status)
	NodeCount       uint32
	NodeOkCount     uint32
	NumberOfVirtual uint32
	sync.Mutex
}

// virtualKey
func (r *HashRing) virtualKey(nodeID string, index uint32) string {
	return "node#" + nodeID + "#" + strconv.FormatUint(uint64(index), 10)
}

// hashKey
func (r *HashRing) hashKey(key string) string {
	return utils.CalcHash([]byte(key))
}

// hashTOCRC32
func (r *HashRing) hashToCRC32(hashInString string) uint32 {
	return utils.CalcCRC32([]byte(hashInString))
}

// CalcIndex
func (r *HashRing) CalcIndex(key string) uint32 {
	return r.hashToCRC32(r.hashKey(key))
}

// AddNode
func (r *HashRing) AddNode(node *Node) {

	r.Lock()

	defer r.Unlock()

	var numberOfNode uint32 = 1
	if r.NumberOfVirtual > 0 {
		numberOfNode = r.NumberOfVirtual
	}

	var i uint32
	for i = 0; i < numberOfNode; i++ {
		index := r.CalcIndex(r.virtualKey(node.ID, i))
		r.VRing.Insert(&VNode{Index: index, NodeID: node.ID})
	}

	r.Nodes.Store(node.ID, node)
	r.NodeStatus.Store(node.ID, false)

	r.NRing.Insert(node)

	r.NodeCount++
}

// RemoveNode
func (r *HashRing) RemoveNode(nodeID string) bool {
	r.Lock()
	defer r.Unlock()

	val, ok := r.Nodes.Load(nodeID)
	if !ok {
		return true
	}
	node := val.(*Node)

	var numberOfNode uint32 = 1
	if r.NumberOfVirtual > 0 {
		numberOfNode = r.NumberOfVirtual
	}

	var i uint32
	for i = 0; i < numberOfNode; i++ {
		index := r.CalcIndex(r.virtualKey(node.ID, i))
		r.VRing.Delete(&VNode{Index: index, NodeID: node.ID})
	}

	r.Nodes.Delete(node.ID)
	r.NodeStatus.Delete(node.ID)

	r.NRing.Delete(node)

	r.NodeCount--

	return true
}

func (r *HashRing) Node(ID string) *Node {
	if node, ok := r.Nodes.Load(ID); ok {
		return node.(*Node)
	}
	return nil
}

func (r *HashRing) IsOnline(ID string) bool {
	online, ok := r.NodeStatus.Load(ID)
	return ok && online.(bool)
}

func (r *HashRing) SetOffline(ID string) {
	r.Lock()
	defer r.Unlock()

	if online, ok := r.NodeStatus.Load(ID); ok && online.(bool) {
		r.NodeStatus.Store(ID, false)
		r.NodeOkCount--
	}
}

func (r *HashRing) SetOnline(ID string) {
	r.Lock()
	defer r.Unlock()

	if online, ok := r.NodeStatus.Load(ID); ok && !online.(bool) {
		r.NodeOkCount++
		r.NodeStatus.Store(ID, true)
	}
}

func (r *HashRing) RandomGetNodes(num int) []*Node {

	if r.NodeOkCount <= 0 {
		return nil
	}

	if r.NodeOkCount < uint32(num) {
		num = int(r.NodeOkCount)
	}

	nodes := make([]*Node, num)

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
			nodes[i] = node.(*Node)
		}
	}

	return nodes
}

// GetNode
// @params key
func (r *HashRing) GetNode(key string) (uint32, string) {
	keyIndex := r.CalcIndex(key)
	return r.GetNodeByIndex(keyIndex)
}

// GetNodeMissNodeID get node excluded given NodeIDs
// @params key
func (r *HashRing) GetNodeExcludedNodeIDs(key string, NodeIDs []string) (uint32, string) {

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
	//	node := value.(*Node)
	//	if !utils.StrInSlices(NodeIDs, node.ID) {
	//		tmpRing.AddNode(&Node{
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
func (r *HashRing) GetNodeUpDownNodes(NodeID string) (string, string) {
	online, ok := r.NodeStatus.Load(NodeID)
	if NodeID == "" || !ok || !online.(bool) || r.NodeCount <= 0 {
		return "", ""
	}

	if r.NRing.Len() <= 1 {
		return "", ""
	}

	up := r.NRing.Max().(*Node).ID
	down := r.NRing.Min().(*Node).ID

	r.NRing.Descend(&Node{ID: NodeID}, func(item rbtree.Item) bool {
		if utils.CalcCRC32([]byte(NodeID)) == utils.CalcCRC32([]byte(item.(*Node).ID)) {
			return true
		}
		up = item.(*Node).ID
		return false
	})

	r.NRing.Ascend(&Node{ID: NodeID}, func(item rbtree.Item) bool {
		if utils.CalcCRC32([]byte(NodeID)) == utils.CalcCRC32([]byte(item.(*Node).ID)) {
			return true
		}
		down = item.(*Node).ID
		return false
	})

	return up, down
}

// GetNodeByIndex
// @params keyIndex
func (r *HashRing) GetNodeByIndex(keyIndex uint32) (uint32, string) {

	if r.VRing.Len() <= 0 {
		return 0, ""
	}

	minVNodeOfRing := r.VRing.Min().(*VNode)

	vNode := minVNodeOfRing

	r.VRing.Ascend(&VNode{Index: keyIndex}, func(item rbtree.Item) bool {
		vNode = item.(*VNode)
		if online, ok := r.NodeStatus.Load(vNode.NodeID); ok && !online.(bool) {
			return true
		}
		return false
	})

	if online, ok := r.NodeStatus.Load(vNode.NodeID); ok && !online.(bool) {
		r.VRing.Ascend(minVNodeOfRing, func(item rbtree.Item) bool {
			vNode = item.(*VNode)
			if online, ok := r.NodeStatus.Load(vNode.NodeID); ok && !online.(bool) {
				return true
			}
			return false
		})
	}

	return vNode.Index, vNode.NodeID
}

// PrintNodes print all non-virtual nodes
func (r *HashRing) PrintNodes() {

	if r.NodeCount <= 0 {
		fmt.Println("nodes is empty")
		return
	}

	r.Nodes.Range(func(key, value interface{}) bool {
		node := value.(*Node)
		fmt.Println(strings.Repeat("=", 30))
		fmt.Println("NodeID:", node.ID)
		fmt.Println("NodeHost:", node.Host)
		fmt.Println("NodeKey :", node.nodeKey())
		fmt.Println()
		return true
	})
}

// TraversalVRing traverse virtual rbtree
func (r *HashRing) TraversalVRing() {
	r.VRing.Ascend(r.VRing.Min(), func(item rbtree.Item) bool {
		fmt.Printf("vNode %d => %s\n", item.(*VNode).Index, item.(*VNode).NodeID)
		return true
	})
}

// TraversalNRing traverse non-virtual rbtree
func (r *HashRing) TraversalNRing() {
	r.NRing.Ascend(r.NRing.Min(), func(item rbtree.Item) bool {
		fmt.Printf("Node %d => %s\n", utils.CalcCRC32([]byte(item.(*Node).ID)), item.(*Node).ID)
		return true
	})
}

// NewHashRing
func New(numOfVNode uint32) *HashRing {
	r := new(HashRing)
	r.Nodes = new(sync.Map)
	r.NodeStatus = new(sync.Map)
	r.NodeCount = 0
	r.NumberOfVirtual = numOfVNode

	r.VRing = rbtree.New()
	r.NRing = rbtree.New()
	return r
}
