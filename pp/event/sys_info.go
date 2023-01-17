package event

import (
	"context"
	"time"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/pp/p2pserver"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
)

var myClock = clock.NewClock()
var job clock.Job

// ReqGetHDInfo
func ReqGetHDInfo(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqGetHDInfo
	if requests.UnmarshalData(ctx, &target) {

		if setting.P2PAddress == target.P2PAddress {
			p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.RspGetHDInfoData(), header.RspGetHDInfo)
		} else {
			p2pserver.GetP2pServer(ctx).TransferSendMessageToPPServByP2pAddress(ctx, target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}

// RspGetHDInfo
func RspGetHDInfo(ctx context.Context, conn core.WriteCloser) {

	p2pserver.GetP2pServer(ctx).TransferSendMessageToSPServer(ctx, core.MessageFromContext(ctx))
}

func reportDHInfo(ctx context.Context) func() {
	return func() {
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.RspGetHDInfoData(), header.RspGetHDInfo)
	}
}

func reportDHInfoToPP(ctx context.Context) {
	p2pserver.GetP2pServer(ctx).SendMessage(ctx, p2pserver.GetP2pServer(ctx).GetPpConn(), requests.RspGetHDInfoData(), header.RspGetHDInfo)
}

func startReportDHInfo(ctx context.Context) {
	if job != nil {
		job.Cancel()
	}
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.RspGetHDInfoData(), header.RspGetHDInfo)
	job, _ = myClock.AddJobRepeat(time.Second*setting.REPORTDHTIME, 0, reportDHInfo(ctx))
}

// GetCapacity GetCapacity
//func GetCapacity(reqID string, w http.ResponseWriter) {
//	if setting.CheckLogin() {
//		network.SendMessage(client.ppConn, requests.ReqGetCapacityData(reqID), header.ReqGetCapacity)
//		storeResponseWriter(reqID, w)
//	} else {
//		notLogin(w)
//	}
//}

// ReqGetCapacity
//func ReqGetCapacity(ctx context.Context, conn core.WriteCloser) {
//	network.TransferSendMessageToSPServer(core.MessageFromContext(ctx))
//}

// RspGetCapacity
//func RspGetCapacity(ctx context.Context, conn core.WriteCloser) {
//	var target protos.RspGetCapacity
//	if requests.UnmarshalData(ctx, &target) {
//		if target.P2PAddress == setting.P2PAddress {
//			if target.Result.State == protos.ResultState_RES_SUCCESS {
//				fmt.Println("action  successfully", target.Result.Msg)
//			} else {
//				fmt.Println("action  failed", target.Result.Msg)
//			}
//			putData(target.ReqId, HTTPGetCapacity, &target)
//		} else {
//			network.TransferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
//		}
//	}
//}
