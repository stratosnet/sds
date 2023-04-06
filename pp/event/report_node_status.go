package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/utils"
)

// RspReportNodeStatus
func RspReportNodeStatus(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspReportNodeStatus
	if err := VerifyMessage(ctx, header.RspReportNodeStatus, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	utils.DebugLogf("get RspReportNodeStatus RSP = %v", target.GetPpstate())
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.ErrorLog(target.Result.Msg)
		return
	}

	if target.Ppstate == int32(protos.PPState_SUSPEND) {
		network.GetPeer(ctx).RunFsm(ctx, network.EVENT_RCV_SUSPENDED_STATE)
	}

	if state := network.GetPeer(ctx).GetStateFromFsm(); state.Id == network.STATE_REGISTERING {
		utils.DebugLog("@#@#@#@#@#@#@#@#@#@#@#@#@#@#")
		network.GetPeer(ctx).RunFsm(ctx, network.EVENT_RCV_RSP_FIRST_STATUS)
	}
}
