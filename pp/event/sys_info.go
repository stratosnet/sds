package event

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"

	"github.com/alex023/clock"
)

var myClock = clock.NewClock()
var job clock.Job

// ReqGetHDInfo
func ReqGetHDInfo(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqGetHDInfo
	if types.UnmarshalData(ctx, &target) {

		if setting.P2PAddress == target.P2PAddress {
			peers.SendMessageToSPServer(types.RspGetHDInfoData(peers.GetDHInfo()), header.RspGetHDInfo)
		} else {
			peers.TransferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}

// RspGetHDInfo
func RspGetHDInfo(ctx context.Context, conn core.WriteCloser) {

	peers.TransferSendMessageToSPServer(core.MessageFromContext(ctx))
}

func reportDHInfo() {
	peers.SendMessageToSPServer(types.RspGetHDInfoData(peers.GetDHInfo()), header.RspGetHDInfo)
}

func reportDHInfoToPP() {
	peers.SendMessage(client.PPConn, types.RspGetHDInfoData(peers.GetDHInfo()), header.RspGetHDInfo)
}

func startReportDHInfo() {
	if job != nil {
		job.Cancel()
	}
	peers.SendMessageToSPServer(types.RspGetHDInfoData(peers.GetDHInfo()), header.RspGetHDInfo)
	job, _ = myClock.AddJobRepeat(time.Second*setting.REPROTDHTIME, 0, reportDHInfo)
}

// GetCapacity GetCapacity
func GetCapacity(reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		peers.SendMessage(client.PPConn, types.ReqGetCapacityData(reqID), header.ReqGetCapacity)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqGetCapacity
func ReqGetCapacity(ctx context.Context, conn core.WriteCloser) {
	peers.TransferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspGetCapacity
func RspGetCapacity(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspGetCapacity
	if types.UnmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				fmt.Println("action  successfully", target.Result.Msg)
			} else {
				fmt.Println("action  failed", target.Result.Msg)
			}
			putData(target.ReqId, HTTPGetCapacity, &target)
		} else {
			peers.TransferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}
