package event

import (
	"context"
	"time"

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

// ReqPpLatencyCheck Request latency measurement to a pp
func ReqPpLatencyCheck(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqPpLatencyCheck
	if err := VerifyMessage(ctx, header.ReqPpLatencyCheck, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		utils.ErrorLog("unmarshal error")
		return
	}
	response := &protos.RspPpLatencyCheck{
		P2PAddressPp: setting.Config.P2PAddress,
	}
	_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, response, header.RspPpLatencyCheck)
}

func RspPpLatencyCheck(ctx context.Context, _ core.WriteCloser) {
	var target protos.RspPpLatencyCheck
	utils.DebugLog("RspLantencyCheck")

	if err := VerifyMessage(ctx, header.RspPpLatencyCheck, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	rspTime := time.Now().UnixNano()
	var response protos.RspPpLatencyCheck
	if !requests.UnmarshalData(ctx, &response) {
		pp.ErrorLog(ctx, "unmarshal error")
		return
	}
	peer := p2pserver.GetP2pServer(ctx).GetPPByP2pAddress(ctx, response.P2PAddressPp)
	if peer == nil {
		return
	}
	if start, ok := network.GetPeer(ctx).LoadPingTimeMap(peer.NetworkAddress, false); ok {
		peer.Latency = rspTime - start
		p2pserver.GetP2pServer(ctx).UpdatePP(ctx, peer)
		network.GetPeer(ctx).DeletePingTimeMap(peer.NetworkAddress, false)
	}
}
