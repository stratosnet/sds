package hashring

import (
	"strconv"
	"testing"
)

func TestGetNodeExcludedNodeIDs(t *testing.T) {
	ring := New(10)

	for i := 0; i < 5; i++ {
		id := "ID#" + strconv.FormatInt(int64(i), 10)
		ring.AddNode(&Node{
			ID:   id,
			Host: "",
			Rest: "",
			Data: nil,
		})
		ring.SetOnline(id)
	}

	if ring.NodeOkCount != 5 {
		t.Fatalf("Wrong NodeOkCount [%v] (expected [%v])", ring.NodeOkCount, 5)
	}

	_, nodeId := ring.GetNodeExcludedNodeIDs("some key", []string{"ID#1", "ID#2", "ID#3", "ID#4"}, false)
	expectedId := "ID#0"
	if nodeId != expectedId {
		t.Fatalf("Wrong node ID found [%v] (expected [%v])", nodeId, expectedId)
	}

	if ring.NodeOkCount != 5 {
		t.Fatalf("Wrong NodeOkCount [%v] (expected [%v])", ring.NodeOkCount, 5)
	}

	_, nodeId = ring.GetNodeExcludedNodeIDs("some other key", []string{"ID#1", "ID#2", "ID#3", "ID#4"}, true)
	if nodeId != expectedId {
		t.Fatalf("Wrong node ID found [%v] (expected [%v])", nodeId, expectedId)
	}

	if ring.NodeOkCount != 1 {
		t.Fatalf("Wrong NodeOkCount [%v] (expected [%v])", ring.NodeOkCount, 1)
	}
}
