package test

import (
	"fmt"
	"testing"

	"github.com/stratosnet/sds/tx-client/grpc"
	"github.com/stretchr/testify/require"
)

func TestQuery(t *testing.T) {
	initGrpcTestSettings()
	testQueryMetaNode(t)
}

func testQueryMetaNode(t *testing.T) {
	fmt.Println("------------------ testQueryMetaNode() start ------------------")
	metaNode, err := grpc.QueryMetaNode(initMetaNodeNetworkAddr)
	require.NoError(t, err)
	fmt.Println("MetaNode = ", metaNode)
}
