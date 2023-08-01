package event

import (
	"context"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/utils"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/requests"
)

var myClock = clock.NewClock() //nolint:unused
var job clock.Job              //nolint:unused

func ReqGetHDInfo(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqGetHDInfo
	if err := VerifyMessage(ctx, header.ReqGetHDInfo, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if requests.UnmarshalData(ctx, &target) {

		if p2pserver.GetP2pServer(ctx).GetP2PAddress() == target.P2PAddress {
			p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.RspGetHDInfoData(p2pserver.GetP2pServer(ctx).GetP2PAddress()), header.RspGetHDInfo)
		} else {
			p2pserver.GetP2pServer(ctx).TransferSendMessageToPPServByP2pAddress(ctx, target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}

func RspGetHDInfo(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspGetHDInfo
	if err := VerifyMessage(ctx, header.RspGetHDInfo, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
	}
	p2pserver.GetP2pServer(ctx).TransferSendMessageToSPServer(ctx, core.MessageFromContext(ctx))
}

//nolint:unused
func reportDHInfo(ctx context.Context) func() {
	return func() {
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.RspGetHDInfoData(p2pserver.GetP2pServer(ctx).GetP2PAddress()), header.RspGetHDInfo)
	}
}

func reportDHInfoToPP(ctx context.Context) {
	_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, p2pserver.GetP2pServer(ctx).GetPpConn(), requests.RspGetHDInfoData(p2pserver.GetP2pServer(ctx).GetP2PAddress()), header.RspGetHDInfo)
}
