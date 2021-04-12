package main

import (
	"fmt"
	"github.com/stratosnet/sds/utils/hashring"
	"strconv"
)

func main() {

	// new net
	ring := hashring.New(20)

	nodes := []*hashring.Node{
		{ID: "php", Host: "localhost:8011"},
		{ID: "java", Host: "localhost:8012"},
		{ID: "golang", Host: "localhost:8013"},
		{ID: "objectc", Host: "localhost:8014"},
		{ID: "clang", Host: "localhost:8015"},
		{ID: "c++", Host: "localhost:8016"},
		{ID: "python", Host: "localhost:8017"},
	}

	// add node
	for i := 0; i < len(nodes); i++ {
		ring.AddNode(nodes[i])
	}

	ring.SetOnline("php")
	ring.SetOnline("java")
	ring.SetOnline("golang")
	ring.SetOnline("c++")

	for _, n := range ring.RandomGetNodes(3) {
		fmt.Printf("id = %s \n", n.ID)
	}

	//testSearch(ring)
	//
	//// delete node
	//ring.RemoveNode(nodes[2].ID)
	//
	//testSearch(ring)
	//
	////
	//testUpDown(ring, "java")
}

func testUpDown(ring *hashring.HashRing, id string) {
	fmt.Println(ring.GetNodeUpDownNodes(id))
}

func testSearch(ring *hashring.HashRing) {

	hitNodeMap := make(map[string]int)

	fmt.Println("node list:")
	ring.PrintNodes()
	fmt.Println()

	fmt.Println("virtual node binary tree:")
	ring.TraversalVRing()

	fmt.Println()

	ring.SetOffline("java")
	fmt.Println("traverse process:")
	for i := 0; i < 100; i++ {

		key := "key:" + strconv.Itoa(i)
		keyIndex := ring.CalcIndex(key)

		// node info
		//index, nodeID := ring.GetNode(key)

		// get node by index
		//index, nodeID := ring.GetNodeByIndex(keyIndex)

		// excluded node
		index, nodeID := ring.GetNodeExcludedNodeIDs(key, []string{"java", "php"})

		if index > 0 {
			fmt.Printf("%s : %d <==> [%d]:{ id: %s}\n", key, keyIndex, index, nodeID)
			hitNodeMap[nodeID]++
		}
	}

	fmt.Println()

	fmt.Println("hit node map:")
	fmt.Println(hitNodeMap)
	fmt.Println("================================================")
}
