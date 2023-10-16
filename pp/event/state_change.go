package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

func ReqStateChange(ctx context.Context, conn core.WriteCloser) {
	req := &protos.ReqStateChangePP{
		P2PAddress: p2pserver.GetP2pServer(ctx).GetP2PAddress(),
	}
	_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, req, header.ReqStateChangePP)
}

func RspStateChange(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspStateChangePP
	if err := VerifyMessage(ctx, header.RspStateChangePP, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if target.UpdateState != uint32(protos.PPState_OFFLINE) {
		utils.Log("State change hasn't been completed")
		return
	}
	if setting.Config.Node.AutoStart {
		utils.Log("State change has been completed, will start registering automatically")
		network.GetPeer(ctx).RunFsm(ctx, network.EVENT_RCV_STATE_OFFLINE)
		return
	}
	utils.Log("State change has been completed, please register you node manually")
}
