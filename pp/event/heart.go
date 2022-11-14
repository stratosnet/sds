package event

import (
	"context"
	"sync"
	"time"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/p2pserver"
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
	if p2pserver.GetP2pServer(ctx).GetConnectionName(conn) != p2pserver.GetP2pServer(ctx).GetSpName() {
		//utils.DebugLogf("====== not sending latency check %v ======", client.GetConnectionName(conn))
		return
	}
	//utils.DebugLogf("====== sending latency check %v ======", client.GetConnectionName(conn))

	p2pserver.GetP2pServer(ctx).ClearBufferedSpConns()
	network.GetPeer(ctx).ClearPingTimeSPMap()
	// clear optSp before ping sp list
	summary.optSp = OptimalSp{}
	go SendLatencyCheckMessageToSPList(ctx)
	myClockLatency := clock.NewClock()
	myClockLatency.AddJobRepeat(time.Second*utils.LatencyCheckSpListTimeout, 1, connectAndRegisterToOptSp(ctx))
}

// SendLatencyCheckMessageToSPList
func SendLatencyCheckMessageToSPList(ctx context.Context) {
	utils.DebugLogf("[SP_LATENCY_CHECK] SendHeartbeatToSPList, num of SPs: %v", len(setting.Config.SPList))
	if len(setting.Config.SPList) < 2 {
		utils.ErrorLog("there are not enough SP nodes in the config file")
		return
	}
	for i := 0; i < len(setting.Config.SPList); i++ {
		selectedSP := setting.Config.SPList[i]
		checkSingleSpLatency(ctx, selectedSP.NetworkAddress, false)
	}
}

func checkSingleSpLatency(ctx context.Context, server string, heartbeat bool) {
	if !p2pserver.GetP2pServer(ctx).SpConnValid() {
		utils.DebugLog("SP latency check skipped until connection to SP is recovered")
		return
	}
	utils.DebugLog("[SP_LATENCY_CHECK] SendHeartbeat(", server, ", req, header.ReqHeartbeat)")
	var spConn *cf.ClientConn
	var err error
	if p2pserver.GetP2pServer(ctx).GetSpName() != server {
		spConn, err = p2pserver.GetP2pServer(ctx).NewClient(ctx, server, heartbeat)
		if err != nil {
			utils.DebugLogf("failed to connect to server %v: %v", server, utils.FormatError(err))
		}
	} else {
		utils.DebugLog("Checking latency for working SP ", server)
		spConn = p2pserver.GetP2pServer(ctx).GetSpConn()
	}
	//defer spConn.Close()
	if spConn != nil {
		start := time.Now().UnixNano()
		network.GetPeer(ctx).StorePingTimeSPMap(server, start)
		pb := &protos.ReqLatencyCheck{
			HbType:           protos.HeartbeatType_LATENCY_CHECK,
			P2PAddressPp:     setting.P2PAddress,
			NetworkAddressSp: server,
		}
		p2pserver.GetP2pServer(ctx).SendMessage(ctx, spConn, pb, header.ReqLatencyCheck)
		if p2pserver.GetP2pServer(ctx).GetSpName() != server {
			p2pserver.GetP2pServer(ctx).StoreBufferedSpConn(spConn)
		}
	}
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
	p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, response, header.RspLatencyCheck)
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
		peer := p2pserver.GetP2pServer(ctx).GetPPByP2pAddress(ctx, response.P2PAddressPp)
		if peer == nil {
			return
		}
		if start, ok := network.GetPeer(ctx).LoadPingTimeSPMap(peer.NetworkAddress); ok {
			peer.Latency = rspTime - start
			p2pserver.GetP2pServer(ctx).UpdatePP(ctx, peer)
			// delete the KV from pingTimePPMap
			network.GetPeer(ctx).DeletePingTimePPMap(peer.NetworkAddress)
		}
	} else if response.HbType == protos.HeartbeatType_LATENCY_CHECK {
		if start, ok := network.GetPeer(ctx).LoadPingTimeSPMap(response.NetworkAddressSp); ok {
			timeCost := rspTime - start
			go updateOptimalSp(ctx, timeCost, &response, &summary.optSp)
			// delete the KV from pingTimeSPMap
			network.GetPeer(ctx).DeletePingTimePPMap(response.NetworkAddressSp)
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
		spConnsToClose := p2pserver.GetP2pServer(ctx).GetBufferedSpConns()
		pp.DebugLogf(ctx, "closing %v spConns", len(spConnsToClose))
		for _, spConn := range spConnsToClose {
			if p2pserver.GetP2pServer(ctx).SpConnValid() && spConn.GetName() == p2pserver.GetP2pServer(ctx).GetSpName() {
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
		p2pserver.GetP2pServer(ctx).ConfirmOptSP(ctx, summary.optSp.NetworkAddr)
		summary.mtx.Unlock()
	}
}
