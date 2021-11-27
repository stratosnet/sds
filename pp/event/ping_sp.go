package event

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

type OptimalSp struct {
	NetworkAddr        string
	SpResponseTimeCost int64
}

type PingRspSummary struct {
	optSp *OptimalSp
	mtx   sync.Mutex
}

var (
	summary = &PingRspSummary{}
)

func ReqPingSpList(ctx context.Context, conn core.WriteCloser) {
	// clear optSp before ping sp list
	summary.optSp = &OptimalSp{}
	go peers.SendPingMessageToSPServers()
	myClock = clock.NewClock()
	myClock.AddJobRepeat(time.Second*utils.PingSpListTimeout, 1, connectAndRegisterToOptSp)
}

func RspPingSpList(ctx context.Context, _ core.WriteCloser) {
	rspTime := time.Now().UnixNano()
	response := &protos.RspPing{}
	if !requests.UnmarshalData(ctx, response) {
		utils.ErrorLog("unmarshal error")
		return
	}
	utils.DebugLogf("received response of PingMsg from SP %v", response)
	updateOptimalSp(rspTime, response, summary.optSp)
}

func updateOptimalSp(rspTime int64, rsp *protos.RspPing, optSp *OptimalSp) {
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
	}
}

func connectAndRegisterToOptSp() {
	summary.mtx.Lock()
	// clear optSp before ping sp list
	if len(summary.optSp.NetworkAddr) == 0 {
		utils.ErrorLog("Optimal Sp isn't found")
		summary.mtx.Unlock()
		return
	}
	peers.ConnectAndRegisterToOptSP(summary.optSp.NetworkAddr)
	summary.mtx.Unlock()
}
