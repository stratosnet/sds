package event

import (
	"context"
	"time"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/msg/header"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/sds-msg/protos"
)

// RspSpLatencyCheck message RspSpLatencyCheck's handler
func RspSpLatencyCheck(ctx context.Context, _ core.WriteCloser) {
	var target protos.RspSpLatencyCheck

	if err := VerifyMessage(ctx, header.RspSpLatencyCheck, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}

	rspTime := time.Now().UnixNano()
	var response protos.RspSpLatencyCheck
	if !requests.UnmarshalData(ctx, &response) {
		pp.ErrorLog(ctx, "unmarshal error")
		return
	}

	if start, ok := network.GetPeer(ctx).LoadPingTimeMap(response.NetworkAddressSp); ok {
		timeCost := rspTime - start
		updateOptimalSp(ctx, timeCost, &response)
		network.GetPeer(ctx).DeletePingTimeMap(response.NetworkAddressSp)
	}
}

func updateOptimalSp(ctx context.Context, timeCost int64, rsp *protos.RspSpLatencyCheck) {
	utils.DebugLogf("Received latency %vns from SP %v", timeCost, rsp.NetworkAddressSp)
	if rsp.P2PAddressPp != setting.Config.Keys.P2PAddress || len(rsp.P2PAddressPp) == 0 {
		// invalid response containing unknown PP p2pAddr
		return
	}
	if timeCost <= 0 {
		return
	}

	candidateSp := network.CandidateSp{
		NetworkAddr:        rsp.NetworkAddressSp,
		SpResponseTimeCost: timeCost,
	}
	network.GetPeer(ctx).UpdateSpCandidateList(candidateSp)
}
