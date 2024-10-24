package hashring

import (
	"fmt"
	"sync"

	"github.com/stratosnet/sds/framework/utils"
)

type Node struct {
	Host      string
	Rest      string
	Data      *sync.Map
	DiskUsage float64
	Weight    float64

	id          string
	onlineIndex int // -1 if node is offline
}

func (n *Node) GetID() string {
	if n == nil {
		return ""
	}
	return n.id
}

// SetID will only set the ID if it's missing
func (n *Node) SetID(ID string) {
	if n == nil {
		return
	}
	if n.id == "" {
		n.id = ID
	}
}

func (n *Node) SetDiskUsage(diskSize, freeDisk uint64) {
	if n == nil {
		return
	}
	if diskSize <= 0 || freeDisk <= 0 {
		n.DiskUsage = 1
	}
	n.DiskUsage = float64(diskSize-freeDisk) / float64(diskSize)
}

func (n *Node) IsOnline() bool {
	return n != nil && n.onlineIndex > -1
}

func (n *Node) String() string {
	if n == nil {
		return "nil"
	}
	return fmt.Sprintf("ID=%v Online=%v Weight=%v Host=%v Rest=%v DiskUsage=%v",
		n.id, n.IsOnline(), n.Weight, n.Host, n.Rest, n.DiskUsage)
}

type nodeKey struct {
	id     string
	weight float64
}

func (k nodeKey) Weight() float64 {
	return k.weight
}

type HashRing struct {
	nodes       map[string]*Node     // key = id
	onlineNodes []utils.WeightedItem // []*nodeKey
	sync.Mutex
}

func (r *HashRing) AddNode(id string, node *Node) {
	r.Lock()
	defer r.Unlock()

	if node == nil {
		return
	}

	if id == "" {
		id = node.id
	}
	node.id = id

	if node.Weight == 0 {
		node.Weight = 1
	}
	node.onlineIndex = -1

	r.nodes[id] = node
}

func (r *HashRing) RemoveNode(ID string) {
	r.Lock()
	defer r.Unlock()

	node, ok := r.nodes[ID]
	if !ok {
		return
	}

	r.removeFromOnlineList(node)
	delete(r.nodes, ID)
}

func (r *HashRing) Node(ID string) *Node {
	r.Lock()
	defer r.Unlock()

	return r.nodes[ID]
}

func (r *HashRing) IsOnline(ID string) bool {
	r.Lock()
	defer r.Unlock()

	return r.nodes[ID].IsOnline()
}

func (r *HashRing) SetOnline(ID string) {
	r.Lock()
	defer r.Unlock()

	node := r.nodes[ID]
	if node == nil || node.IsOnline() {
		return
	}

	node.onlineIndex = len(r.onlineNodes)
	r.onlineNodes = append(r.onlineNodes, &nodeKey{
		id:     node.id,
		weight: node.Weight,
	})
}

func (r *HashRing) SetOffline(ID string) {
	r.Lock()
	defer r.Unlock()

	r.removeFromOnlineList(r.nodes[ID])
}

func (r *HashRing) removeFromOnlineList(node *Node) {
	if node == nil {
		return
	}
	if !node.IsOnline() {
		return
	}

	if len(r.onlineNodes) <= 1 {
		r.onlineNodes = nil
	} else {
		lastElement, ok := r.onlineNodes[len(r.onlineNodes)-1].(*nodeKey)
		if !ok {
			return
		}
		if lastElement.id != node.id {
			// node is not the last element. Switch both nodes
			r.onlineNodes[node.onlineIndex] = lastElement
			r.nodes[lastElement.id].onlineIndex = node.onlineIndex
		}
		// Discard last element
		r.onlineNodes = r.onlineNodes[:len(r.onlineNodes)-1]
	}

	node.onlineIndex = -1
}

func (r *HashRing) UpdateNodeDiskUsage(ID string, diskSize, freeDisk uint64) {
	r.Lock()
	defer r.Unlock()

	r.nodes[ID].SetDiskUsage(diskSize, freeDisk)
}

func (r *HashRing) NodeCount() uint32 {
	r.Lock()
	defer r.Unlock()

	return uint32(len(r.nodes))
}

func (r *HashRing) NodeOkCount() uint32 {
	r.Lock()
	defer r.Unlock()

	return uint32(len(r.onlineNodes))
}

// RandomNode uses the given seed to select a random online node
// If the seed is empty, it will use a cryptographically secure random seed
func (r *HashRing) RandomNode(seed string) (string, *Node) {
	r.Lock()
	defer r.Unlock()

	_, node := utils.WeightedRandomSelect(r.onlineNodes, seed)
	key, ok := node.(*nodeKey)
	if !ok {
		return "", nil
	}
	selectedID := key.id
	return selectedID, r.nodes[selectedID]
}

// RandomNodeExcludedIDs uses the given seed to select a random online node, while excluding specified nodes
// If the seed is empty, it will use a cryptographically secure random seed
func (r *HashRing) RandomNodeExcludedIDs(excludedIDs []string, seed string) (string, *Node) {
	r.Lock()
	defer r.Unlock()

	exclusionMap := make(map[string]bool)
	for _, exclusion := range excludedIDs {
		exclusionMap[exclusion] = true
	}

	var filteredNodes []utils.WeightedItem
	for _, node := range r.onlineNodes {
		key, ok := node.(*nodeKey)
		if !ok {
			continue
		}
		if !exclusionMap[key.id] {
			filteredNodes = append(filteredNodes, node)
		}
	}

	_, node := utils.WeightedRandomSelect(filteredNodes, seed)
	key, ok := node.(*nodeKey)
	if !ok {
		return "", nil
	}
	selectedID := key.id
	return selectedID, r.nodes[selectedID]
}

// RandomNodes return random nodes from the hashring
// If the seed is empty, it will use a cryptographically secure random seed
func (r *HashRing) RandomNodes(num int, seed string) []*Node {
	r.Lock()
	defer r.Unlock()

	_, nodes := utils.WeightedRandomSelectMultiple(r.onlineNodes, num, seed)

	var selectedNodes []*Node
	for _, node := range nodes {
		key, ok := node.(*nodeKey)
		if !ok {
			continue
		}
		selectedNodes = append(selectedNodes, r.nodes[key.id])
	}
	return selectedNodes
}

func (r *HashRing) String() string {
	r.Lock()
	defer r.Unlock()

	if len(r.nodes) <= 0 {
		return "Empty hashring"
	}
	str := ""
	for _, node := range r.nodes {
		str += fmt.Sprintln(node)
	}
	return str
}

func New() *HashRing {
	return &HashRing{
		nodes:       make(map[string]*Node),
		onlineNodes: nil,
		Mutex:       sync.Mutex{},
	}
}
