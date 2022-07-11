package event

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

type OptimalIndexNode struct {
	NetworkAddr               string
	IndexNodeResponseTimeCost int64
}

type LatencyCheckRspSummary struct {
	optIndexNode OptimalIndexNode
	mtx          sync.Mutex
}

var (
	summary = &LatencyCheckRspSummary{}
)

func ReqHBLatencyCheckIndexNodeList(ctx context.Context, conn core.WriteCloser) {
	if client.GetConnectionName(conn) != client.GetConnectionName(client.IndexNodeConn) {
		//utils.DebugLogf("====== not sending latency check %v ======", client.GetConnectionName(conn))
		return
	}
	//utils.DebugLogf("====== sending latency check %v ======", client.GetConnectionName(conn))
	peers.ClearBufferedIndexNodeConns()
	// clear optIndexNode before ping index node list
	summary.optIndexNode = OptimalIndexNode{}
	go peers.SendLatencyCheckMessageToIndexNodeList()
	myClockLatency := clock.NewClock()
	myClockLatency.AddJobRepeat(time.Second*utils.LatencyCheckIndexNodeListTimeout, 1, connectAndRegisterToOptIndexNode)
}

func RspHBLatencyCheckIndexNodeList(ctx context.Context, _ core.WriteCloser) {
	utils.DebugLog("get Heartbeat RSP")
	rspTime := time.Now().UnixNano()
	var response protos.RspLatencyCheck
	if !requests.UnmarshalData(ctx, &response) {
		utils.ErrorLog("unmarshal error")
		return
	}
	if response.HbType != protos.HeartbeatType_LATENCY_CHECK {
		utils.ErrorLog("invalid response of heartbeat")
		return
	}
	go updateOptimalIndexNode(rspTime, &response, &summary.optIndexNode)
}

func updateOptimalIndexNode(rspTime int64, rsp *protos.RspLatencyCheck, optIndexNode *OptimalIndexNode) {
	summary.mtx.Lock()
	if rsp.P2PAddressPp != setting.Config.P2PAddress || len(rsp.P2PAddressPp) == 0 {
		// invalid response containing unknown PP p2pAddr
		return
	}
	reqTime, err := strconv.ParseInt(rsp.PingTime, 10, 64)
	if err != nil {
		utils.ErrorLog("cannot parse ping time from response")
		return
	}
	timeCost := rspTime - reqTime
	if timeCost <= 0 {
		return
	}
	if len(optIndexNode.NetworkAddr) == 0 || timeCost < optIndexNode.IndexNodeResponseTimeCost {
		// update new index node
		optIndexNode.NetworkAddr = rsp.NetworkAddressIndexNode
		optIndexNode.IndexNodeResponseTimeCost = timeCost
		utils.DebugLogf("New optimal Index Node is %v", optIndexNode)
	}
	summary.mtx.Unlock()
}

func connectAndRegisterToOptIndexNode() {
	summary.mtx.Lock()
	// clear buffered IndexNodeConn
	indexNodeConnsToClose := peers.GetBufferedIndexNodeConns()
	utils.DebugLogf("closing %v indexNodeConns", len(indexNodeConnsToClose))
	for _, indexNodeConn := range indexNodeConnsToClose {
		if indexNodeConn.GetName() == client.IndexNodeConn.GetName() {
			utils.DebugLogf("indexNodeConn %v in connection, not closing it", indexNodeConn.GetName())
			continue
		}
		utils.DebugLogf("closing indexNodeConn %v", indexNodeConn.GetName())
		indexNodeConn.Close()
	}
	// clear optIndexNode before ping index node list
	if len(summary.optIndexNode.NetworkAddr) == 0 {
		utils.ErrorLog("Optimal Index Node isn't found")
		summary.mtx.Unlock()
		return
	}
	peers.ConfirmOptIndexNode(summary.optIndexNode.NetworkAddr)
	summary.mtx.Unlock()
}
