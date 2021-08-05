package event

import (
	"context"
	"fmt"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/setting"
	"net/http"
	"time"

	"github.com/alex023/clock"
)

var myClock = clock.NewClock()
var job clock.Job

// ReqGetHDInfo
func ReqGetHDInfo(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqGetHDInfo
	if unmarshalData(ctx, &target) {

		if setting.P2PAddress == target.P2PAddress {
			SendMessageToSPServer(rspGetHDInfoData(), header.RspGetHDInfo)
		} else {
			transferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}

// RspGetHDInfo
func RspGetHDInfo(ctx context.Context, conn core.WriteCloser) {

	transferSendMessageToSPServer(core.MessageFromContext(ctx))
}

func reportDHInfo() {
	SendMessageToSPServer(rspGetHDInfoData(), header.RspGetHDInfo)
}

func reportDHInfoToPP() {
	sendMessage(client.PPConn, rspGetHDInfoData(), header.RspGetHDInfo)
}

func startReportDHInfo() {
	if job != nil {
		job.Cancel()
	}
	SendMessageToSPServer(rspGetHDInfoData(), header.RspGetHDInfo)
	job, _ = myClock.AddJobRepeat(time.Second*setting.REPROTDHTIME, 0, reportDHInfo)
}

// GetCapacity GetCapacity
func GetCapacity(reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		sendMessage(client.PPConn, reqGetCapacityData(reqID), header.ReqGetCapacity)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqGetCapacity
func ReqGetCapacity(ctx context.Context, conn core.WriteCloser) {
	transferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspGetCapacity
func RspGetCapacity(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspGetCapacity
	if unmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				fmt.Println("action  successfully", target.Result.Msg)
			} else {
				fmt.Println("action  failed", target.Result.Msg)
			}
			putData(target.ReqId, HTTPGetCapacity, &target)
		} else {
			transferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}
