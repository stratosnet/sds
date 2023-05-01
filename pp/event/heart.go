package event

import (
	"context"
	"math/rand"
	"sort"
	"sync"
	"time"

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

type candidateSp struct {
	networkAddr        string
	spResponseTimeCost int64
}

var (
	candidateSps []candidateSp
	mtx          sync.Mutex
)

// ReqSpLatencyCheck is called on every client connection, after utils.LatencyCheckSpListInterval seconds. For connections to SP nodes, it triggers the SP latency check
func ReqSpLatencyCheck(ctx context.Context, conn core.WriteCloser) {
	if p2pserver.GetP2pServer(ctx).GetConnectionName(conn) != p2pserver.GetP2pServer(ctx).GetSpName() {
		//utils.DebugLogf("====== not sending latency check %v ======", client.GetConnectionName(conn))
		return
	}
	//utils.DebugLogf("====== sending latency check %v ======", client.GetConnectionName(conn))

	p2pserver.GetP2pServer(ctx).ClearBufferedSpConns()
	network.GetPeer(ctx).ClearPingTimeMap(true)

	// clear list of candidates before pinging sp list
	mtx.Lock()
	defer mtx.Unlock()
	candidateSps = nil

	go sendLatencyCheckMessageToSPList(ctx)
	go func() {
		time.Sleep(time.Second * utils.LatencyCheckSpListTimeout)
		selectAndConnectToOptSp(ctx)
	}()
}

func sendLatencyCheckMessageToSPList(ctx context.Context) {
	utils.DebugLogf("[SP_LATENCY_CHECK] SendHeartbeatToSPList, num of SPs: %v", len(setting.Config.SPList))
	if len(setting.Config.SPList) < 2 {
		utils.ErrorLog("there are not enough SP nodes in the config file")
		return
	}

	for _, selectedSP := range setting.Config.SPList {
		checkSingleSpLatency(ctx, selectedSP.NetworkAddress)
	}
}

func checkSingleSpLatency(ctx context.Context, server string) {
	if !p2pserver.GetP2pServer(ctx).SpConnValid() {
		utils.DebugLog("SP latency check skipped until connection to SP is recovered")
		return
	}
	utils.DebugLog("[SP_LATENCY_CHECK] SendHeartbeat(", server, ", req, header.ReqHeartbeat)")
	var spConn *cf.ClientConn
	var err error
	if p2pserver.GetP2pServer(ctx).GetSpName() != server {
		spConn, err = p2pserver.GetP2pServer(ctx).NewClientToAlternativeSp(ctx, server)
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
		network.GetPeer(ctx).StorePingTimeMap(server, start, true)
		pb := &protos.ReqLatencyCheck{
			HbType:           protos.HeartbeatType_LATENCY_CHECK,
			P2PAddressPp:     p2pserver.GetP2pServer(ctx).GetP2PAddress(),
			NetworkAddressSp: server,
		}
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, spConn, pb, header.ReqLatencyCheck)
		if p2pserver.GetP2pServer(ctx).GetSpName() != server {
			p2pserver.GetP2pServer(ctx).StoreBufferedSpConn(spConn)
		}
	}
}

// ReqLatencyCheckToPp Request latency measurement from a pp and to a pp
func ReqLatencyCheckToPp(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqLatencyCheck
	if err := VerifyMessage(ctx, header.ReqLatencyCheck, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
	}
	if !requests.UnmarshalData(ctx, &target) {
		utils.ErrorLog("unmarshal error")
		return
	}
	response := &protos.RspLatencyCheck{
		HbType:       target.HbType,
		P2PAddressPp: setting.Config.P2PAddress,
	}
	_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, response, header.RspLatencyCheck)
}

func RspLatencyCheck(ctx context.Context, _ core.WriteCloser) {
	var target protos.RspLatencyCheck
	if err := VerifyMessage(ctx, header.RspLatencyCheck, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
	}
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
		if start, ok := network.GetPeer(ctx).LoadPingTimeMap(peer.NetworkAddress, false); ok {
			peer.Latency = rspTime - start
			p2pserver.GetP2pServer(ctx).UpdatePP(ctx, peer)
			network.GetPeer(ctx).DeletePingTimeMap(peer.NetworkAddress, false)
		}
	} else if response.HbType == protos.HeartbeatType_LATENCY_CHECK {
		if start, ok := network.GetPeer(ctx).LoadPingTimeMap(response.NetworkAddressSp, true); ok {
			timeCost := rspTime - start
			go updateOptimalSp(timeCost, &response)
			network.GetPeer(ctx).DeletePingTimeMap(response.NetworkAddressSp, true)
		}
	}
}

func updateOptimalSp(timeCost int64, rsp *protos.RspLatencyCheck) {
	utils.DebugLogf("Received latency %vns from SP %v", timeCost, rsp.NetworkAddressSp)
	if rsp.P2PAddressPp != setting.Config.P2PAddress || len(rsp.P2PAddressPp) == 0 {
		// invalid response containing unknown PP p2pAddr
		return
	}
	if timeCost <= 0 {
		return
	}

	mtx.Lock()
	defer mtx.Unlock()

	for i, candidate := range candidateSps {
		if candidate.networkAddr == rsp.NetworkAddressSp {
			candidateSps[i].spResponseTimeCost = timeCost
			return
		}
	}
	candidateSps = append(candidateSps, candidateSp{
		networkAddr:        rsp.NetworkAddressSp,
		spResponseTimeCost: timeCost,
	})
}

func selectAndConnectToOptSp(ctx context.Context) {
	mtx.Lock()
	defer mtx.Unlock()

	// clear buffered spConn
	spConnsToClose := p2pserver.GetP2pServer(ctx).GetBufferedSpConns()
	pp.DebugLogf(ctx, "closing %v spConns", len(spConnsToClose))
	for _, spConn := range spConnsToClose {
		if p2pserver.GetP2pServer(ctx).SpConnValid() && spConn.GetName() == p2pserver.GetP2pServer(ctx).GetSpName() {
			pp.DebugLogf(ctx, "spConn %v in connection, not closing it", spConn.GetName())
		} else {
			pp.DebugLogf(ctx, "closing spConn %v", spConn.GetName())
			spConn.Close()
		}
	}

	if len(candidateSps) == 0 {
		pp.ErrorLog(ctx, "Couldn't select an optimal SP")
		return
	}

	sort.Slice(candidateSps, func(i, j int) bool {
		return candidateSps[i].spResponseTimeCost < candidateSps[j].spResponseTimeCost
	})
	nSpsConsidered := utils.LatencyCheckTopSpsConsidered // Select from top 3 SPs
	if nSpsConsidered > len(candidateSps) {
		nSpsConsidered = len(candidateSps)
	}
	selectedSp := rand.Intn(nSpsConsidered)
	p2pserver.GetP2pServer(ctx).ConfirmOptSP(ctx, candidateSps[selectedSp].networkAddr)
}
