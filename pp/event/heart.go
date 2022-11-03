package event

import (
	"context"
	"sync"
	"time"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
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
	if client.GetConnectionName(conn) != client.GetConnectionName(client.SPConn) {
		//utils.DebugLogf("====== not sending latency check %v ======", client.GetConnectionName(conn))
		return
	}
	//utils.DebugLogf("====== sending latency check %v ======", client.GetConnectionName(conn))

	peers.ClearBufferedSpConns()
	peers.ClearPingTimeSPMap()
	// clear optSp before ping sp list
	summary.optSp = OptimalSp{}
	go peers.SendLatencyCheckMessageToSPList(ctx)
	myClockLatency := clock.NewClock()
	myClockLatency.AddJobRepeat(time.Second*utils.LatencyCheckSpListTimeout, 1, connectAndRegisterToOptSp(ctx))
}

// Request latency measurement from a pp and to a pp
func ReqLatencyCheckToPp(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqLatencyCheck
	if !requests.UnmarshalData(ctx, &target) {
		utils.ErrorLog("unmarshal error")
		return
	}
	response := &protos.RspLatencyCheck{
		HbType:       target.HbType,
		P2PAddressPp: setting.Config.P2PAddress,
	}
	peers.SendMessage(ctx, conn, response, header.RspLatencyCheck)
	return
}

func RspHBLatencyCheckSpList(ctx context.Context, _ core.WriteCloser) {
	pp.DebugLog(ctx, "get Heartbeat RSP")
	rspTime := time.Now().UnixNano()
	var response protos.RspLatencyCheck
	if !requests.UnmarshalData(ctx, &response) {
		pp.ErrorLog(ctx, "unmarshal error")
		return
	}
	if response.HbType == protos.HeartbeatType_LATENCY_CHECK_PP {
		peer := peers.GetPPByP2pAddress(ctx, response.P2PAddressPp)
		if peer == nil {
			return
		}
		if value, ok := peers.PingTimePPMap.Load(peer.NetworkAddress); ok {
			start := value.(int64)
			peer.Latency = rspTime - start
			peers.UpdatePP(ctx, peer)
			// delete the KV from PingTimePPMap
			peers.PingTimePPMap.Delete(peer.NetworkAddress)
		}
	} else if response.HbType == protos.HeartbeatType_LATENCY_CHECK {
		if value, ok := peers.PingTimeSPMap.Load(response.NetworkAddressSp); ok {
			start := value.(int64)
			timeCost := rspTime - start
			go updateOptimalSp(ctx, timeCost, &response, &summary.optSp)
			// delete the KV from PingTimeSPMap
			peers.PingTimeSPMap.Delete(response.NetworkAddressSp)
		}
	}
}

func updateOptimalSp(ctx context.Context, timeCost int64, rsp *protos.RspLatencyCheck, optSp *OptimalSp) {
	summary.mtx.Lock()
	if rsp.P2PAddressPp != setting.Config.P2PAddress || len(rsp.P2PAddressPp) == 0 {
		// invalid response containing unknown PP p2pAddr
		return
	}
	if timeCost <= 0 {
		return
	}
	if len(optSp.NetworkAddr) == 0 || timeCost < optSp.SpResponseTimeCost {
		// update new sp
		optSp.NetworkAddr = rsp.NetworkAddressSp
		optSp.SpResponseTimeCost = timeCost
		pp.DebugLogf(ctx, "New optimal SP is %v", optSp)
	}
	summary.mtx.Unlock()
}

func connectAndRegisterToOptSp(ctx context.Context) func() {
	return func() {
		summary.mtx.Lock()
		// clear buffered spConn
		spConnsToClose := peers.GetBufferedSpConns()
		pp.DebugLogf(ctx, "closing %v spConns", len(spConnsToClose))
		for _, spConn := range spConnsToClose {
			if spConn.GetName() == client.SPConn.GetName() {
				pp.DebugLogf(ctx, "spConn %v in connection, not closing it", spConn.GetName())
				continue
			}
			pp.DebugLogf(ctx, "closing spConn %v", spConn.GetName())
			spConn.Close()
		}
		// clear optSp before ping sp list
		if len(summary.optSp.NetworkAddr) == 0 {
			pp.ErrorLog(ctx, "Optimal Sp isn't found")
			summary.mtx.Unlock()
			return
		}
		peers.ConfirmOptSP(ctx, summary.optSp.NetworkAddr)
		summary.mtx.Unlock()
	}
}
