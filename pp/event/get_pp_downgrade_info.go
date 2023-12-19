package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/msg/header"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/sds-msg/protos"
)

// RspGetPPDowngradeInfo
func RspGetPPDowngradeInfo(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspGetPPDowngradeInfo
	if err := VerifyMessage(ctx, header.RspGetPPDowngradeInfo, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}

	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		pp.Log(ctx, "failed to query node status, please retry later")
		return
	}
	pp.Logf(ctx, "PP downgrade happened at: %d (heights) ago, at SP node %v, score decreased by %v ", target.DowngradeHeightDeltaToNow, target.SpP2PAddress, target.ScoreDecreased)
}

func ReqGetPPDowngradeInfo(ctx context.Context) error {
	req := requests.ReqDowngradeInfo(ctx)
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, req, header.ReqGetPPDowngradeInfo)
	return nil
}
