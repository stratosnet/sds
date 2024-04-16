package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/msg/header"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/sds-msg/protos"
)

func ReqGetHDInfo(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqGetHDInfo
	if err := VerifyMessage(ctx, header.ReqGetHDInfo, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	if p2pserver.GetP2pServer(ctx).GetP2PAddress().String() != target.P2PAddress {
		return
	}

	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(
		ctx,
		requests.RspGetHDInfoData(p2pserver.GetP2pServer(ctx).GetP2PAddress().String()),
		header.RspGetHDInfo,
	)
}

func RspGetHDInfo(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspGetHDInfo
	if err := VerifyMessage(ctx, header.RspGetHDInfo, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
	}
	p2pserver.GetP2pServer(ctx).TransferSendMessageToSPServer(ctx, core.MessageFromContext(ctx))
}

//nolint:unused
func reportHDInfo(ctx context.Context) func() {
	return func() {
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(
			ctx,
			requests.RspGetHDInfoData(p2pserver.GetP2pServer(ctx).GetP2PAddress().String()),
			header.RspGetHDInfo,
		)
	}
}
