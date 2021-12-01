package event

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

type OptimalSp struct {
	NetworkAddr        string
	SpResponseTimeCost int64
}

type LatencyCheckRspSummary struct {
	optSp OptimalSp
	mtx   sync.Mutex
}

var (
	summary = &LatencyCheckRspSummary{}
)

func ReqHBLatencyCheckSpList(ctx context.Context, conn core.WriteCloser) {
	peers.ClearBufferedSpConns()
	// clear optSp before ping sp list
	summary.optSp = OptimalSp{}
	go peers.SendLatencyCheckMessageToSPList()
	myClockLatency := clock.NewClock()
	myClockLatency.AddJobRepeat(time.Second*utils.LatencyCheckSpListTimeout, 1, connectAndRegisterToOptSp)
}

func RspHBLatencyCheckSpList(ctx context.Context, _ core.WriteCloser) {
	utils.DebugLog("get Heartbeat RSP")
	rspTime := time.Now().UnixNano()
	var response protos.RspHeartbeat
	if !requests.UnmarshalData(ctx, &response) {
		utils.ErrorLog("unmarshal error")
		return
	}
	if response.HbType != protos.HeartbeatType_LATENCY_CHECK {
		utils.ErrorLog("invalid response of heartbeat")
		return
	}
	go updateOptimalSp(rspTime, &response, &summary.optSp)
}

func updateOptimalSp(rspTime int64, rsp *protos.RspHeartbeat, optSp *OptimalSp) {
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
	if len(optSp.NetworkAddr) == 0 || timeCost < optSp.SpResponseTimeCost {
		// update new sp
		optSp.NetworkAddr = rsp.NetworkAddressSp
		optSp.SpResponseTimeCost = timeCost
		utils.DebugLogf("New optimal SP is %v", optSp)
	}
	summary.mtx.Unlock()
}

func connectAndRegisterToOptSp() {
	summary.mtx.Lock()
	// clear buffered spConn
	spConnsToClose := peers.GetBufferedSpConns()
	utils.DebugLogf("closing %v spConns", len(spConnsToClose))
	for _, spConn := range spConnsToClose {
		spConn.Close()
	}
	// clear optSp before ping sp list
	if len(summary.optSp.NetworkAddr) == 0 {
		utils.ErrorLog("Optimal Sp isn't found")
		summary.mtx.Unlock()
		return
	}
	peers.ConnectAndRegisterToOptSP(summary.optSp.NetworkAddr)
	summary.mtx.Unlock()
}

// SendHeartBeat
func SendHeartBeat(ctx context.Context, conn core.WriteCloser) {
	if client.SPConn != nil {
		start := time.Now().UnixNano()
		pb := &protos.ReqHeartbeat{
			HbType:       protos.HeartbeatType_REGULAR_HEARTBEAT,
			P2PAddressPp: setting.P2PAddress,
			PingTime:     strconv.FormatInt(start, 10),
		}
		peers.SendMessage(client.SPConn, pb, header.ReqHeart)
	}
	utils.DebugLog("SP not yet connected, skip heartbeat")
}

// RspHeartBeat - regular heartbeat getting no rsp from sp
func RspHeartBeat(ctx context.Context, conn core.WriteCloser) {
}
