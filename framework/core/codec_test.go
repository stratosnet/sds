package core

import (
	"fmt"
	"testing"

	"github.com/magiconair/properties/assert"

	"github.com/stratosnet/sds/sds-msg/header"

	"github.com/stratosnet/sds/framework/utils"
)

const (
	TEST_NODE_ID    = 133
	TEST_MSG_TYPE_1 = header.MSG_ID_REQ_GET_HDINFO
	TEST_MSG_TYPE_2 = header.MSG_ID_RSP_START_MAINTENANCE
	TEST_MSG_TYPE_3 = header.MSG_ID_REQ_FILE_REPLICA_INFO
)

func TestGenerateReqId(t *testing.T) {

	err := utils.InitIdWorker(TEST_NODE_ID)
	if err != nil {
		t.Fatal(err.Error())
	}
	combineLogger := utils.NewDefaultLogger("./logs/stdout.log", true, true)
	combineLogger.SetLogLevel(utils.Debug)

	testReqId1 := GenerateNewReqId(TEST_MSG_TYPE_1)
	testReqId2 := GenerateNewReqId(TEST_MSG_TYPE_2)
	testReqId3 := GenerateNewReqId(TEST_MSG_TYPE_3)

	fmt.Printf("Test 1: 0x%016X\n", testReqId1)
	assert.Equal(t, testReqId1&0xFF, int64(TEST_MSG_TYPE_1))

	fmt.Printf("Test 2: 0x%016X\n", testReqId2)
	assert.Equal(t, testReqId2&0xFF, int64(TEST_MSG_TYPE_2))

	fmt.Printf("Test 3: 0x%016X\n", testReqId3)
	assert.Equal(t, testReqId3&0xFF, int64(TEST_MSG_TYPE_3))

}
