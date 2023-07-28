package event

import (
	"sync"
	"testing"

	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/setting"
)

const (
	TEST_P2P_ADDR_1     = "TEST_P2P_ADDR_1"
	TEST_P2P_ADDR_2     = "TEST_P2P_ADDR_2"
	TEST_P2P_ADDR_3     = "TEST_P2P_ADDR_3"
	TEST_P2P_ADDR_TEM   = "TEST_P2P_ADDR_TEM"
	TEST_P2P_PubKey_1   = "TEST_P2P_PubKey_1"
	TEST_P2P_PubKey_2   = "TEST_P2P_PubKey_2"
	TEST_P2P_PubKey_3   = "TEST_P2P_PubKey_3"
	TEST_P2P_PubKey_TEM = "TEST_P2P_PubKey_TEM"
)

// TestSpListCheckAddNew insert new sp into the map
func TestSpListCheckAddNew(t *testing.T) {

	testSpListInfo1 := &protos.SPBaseInfo{P2PAddress: TEST_P2P_ADDR_1, P2PPubKey: TEST_P2P_PubKey_1}
	testSpListInfo2 := &protos.SPBaseInfo{P2PAddress: TEST_P2P_ADDR_2, P2PPubKey: TEST_P2P_PubKey_2}
	testSpListInfo3 := &protos.SPBaseInfo{P2PAddress: TEST_P2P_ADDR_3, P2PPubKey: TEST_P2P_PubKey_3}
	testSpMapInfo1 := setting.SPBaseInfo{P2PAddress: TEST_P2P_ADDR_1, P2PPublicKey: TEST_P2P_PubKey_1}
	testSpMapInfo2 := setting.SPBaseInfo{P2PAddress: TEST_P2P_ADDR_2, P2PPublicKey: TEST_P2P_PubKey_2}

	// simulation to SPMap
	setting.SPMap = &sync.Map{}
	setting.SPMap.Store(testSpMapInfo1.P2PAddress, testSpMapInfo1)
	setting.SPMap.Store(testSpMapInfo2.P2PAddress, testSpMapInfo2)

	// simulation to message
	var testList []*protos.SPBaseInfo
	testList = append(testList, testSpListInfo1)
	testList = append(testList, testSpListInfo3)
	testList = append(testList, testSpListInfo2)

	// test check and update the map
	if checkSpListChanged(testList) {
		setting.SPMap = &sync.Map{}
		setting.UpdateSpMap(testList)
	}

	// check result
	i := 0
	setting.SPMap.Range(func(k, v interface{}) bool {
		i++
		return true
	})
	if i != 3 {
		t.Fatal("Test Insert Failed:", i)
	}
}

// TestSpListCheckRemove remove a sp into the map
func TestSpListCheckRemove(t *testing.T) {

	testSpListInfo1 := &protos.SPBaseInfo{P2PAddress: TEST_P2P_ADDR_1, P2PPubKey: TEST_P2P_PubKey_1}
	testSpListInfo3 := &protos.SPBaseInfo{P2PAddress: TEST_P2P_ADDR_3, P2PPubKey: TEST_P2P_PubKey_3}
	testSpMapInfo1 := setting.SPBaseInfo{P2PAddress: TEST_P2P_ADDR_1, P2PPublicKey: TEST_P2P_PubKey_1}
	testSpMapInfo2 := setting.SPBaseInfo{P2PAddress: TEST_P2P_ADDR_2, P2PPublicKey: TEST_P2P_PubKey_2}
	testSpMapInfo3 := setting.SPBaseInfo{P2PAddress: TEST_P2P_ADDR_3, P2PPublicKey: TEST_P2P_PubKey_3}

	// simulation to SPMap
	setting.SPMap = &sync.Map{}
	setting.SPMap.Store(testSpMapInfo3.P2PAddress, testSpMapInfo3)
	setting.SPMap.Store(testSpMapInfo1.P2PAddress, testSpMapInfo1)
	setting.SPMap.Store(testSpMapInfo2.P2PAddress, testSpMapInfo2)

	// simulation to message
	var testList []*protos.SPBaseInfo
	testList = append(testList, testSpListInfo1)
	testList = append(testList, testSpListInfo3)

	// test check and update the map
	if checkSpListChanged(testList) {
		setting.SPMap = &sync.Map{}
		setting.UpdateSpMap(testList)
	}

	// check result
	i := 0
	setting.SPMap.Range(func(k, v interface{}) bool {
		i++
		return true
	})
	if i != 2 {
		t.Fatal("Test Remove Failed")
	}
}

// TestSpListCheckChange change a sp in the map
func TestSpListCheckChange(t *testing.T) {

	testSpListInfo1 := &protos.SPBaseInfo{P2PAddress: TEST_P2P_ADDR_1, P2PPubKey: TEST_P2P_PubKey_1}
	testSpListInfo2 := &protos.SPBaseInfo{P2PAddress: TEST_P2P_ADDR_2, P2PPubKey: TEST_P2P_PubKey_TEM}
	testSpListInfo3 := &protos.SPBaseInfo{P2PAddress: TEST_P2P_ADDR_3, P2PPubKey: TEST_P2P_PubKey_3}
	testSpMapInfo1 := setting.SPBaseInfo{P2PAddress: TEST_P2P_ADDR_1, P2PPublicKey: TEST_P2P_PubKey_1}
	testSpMapInfo2 := setting.SPBaseInfo{P2PAddress: TEST_P2P_ADDR_2, P2PPublicKey: TEST_P2P_PubKey_2}
	testSpMapInfo3 := setting.SPBaseInfo{P2PAddress: TEST_P2P_ADDR_3, P2PPublicKey: TEST_P2P_PubKey_3}

	// simulation to SPMap
	setting.SPMap = &sync.Map{}
	setting.SPMap.Store(testSpMapInfo3.P2PAddress, testSpMapInfo3)
	setting.SPMap.Store(testSpMapInfo1.P2PAddress, testSpMapInfo1)
	setting.SPMap.Store(testSpMapInfo2.P2PAddress, testSpMapInfo2)

	// simulation to message
	var testList []*protos.SPBaseInfo
	testList = append(testList, testSpListInfo2)
	testList = append(testList, testSpListInfo1)
	testList = append(testList, testSpListInfo3)

	// test check and update the map
	if checkSpListChanged(testList) {
		setting.SPMap = &sync.Map{}
		setting.UpdateSpMap(testList)
	}

	// check result
	i := 0
	setting.SPMap.Range(func(k, v interface{}) bool {
		i++
		return true
	})
	if i != 3 {
		t.Fatal("Number of items is wrong")
	}
	testTmpSpInfo, ok := setting.SPMap.Load(testSpListInfo2.P2PAddress)
	if !ok || testTmpSpInfo.(setting.SPBaseInfo).P2PPublicKey != TEST_P2P_PubKey_TEM {
		t.Fatal("Entry is not updated:")
	}

}
