package event

import (
	"context"
	"fmt"
	"github.com/qsnetwork/qsds/framework/spbf"
	"github.com/qsnetwork/qsds/msg/header"
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/pp/client"
	"github.com/qsnetwork/qsds/pp/setting"
	"net/http"
	"time"

	"github.com/alex023/clock"
)

var myClock = clock.NewClock()
var job clock.Job

// ReqGetHDInfo
func ReqGetHDInfo(ctx context.Context, conn spbf.WriteCloser) {
	var target protos.ReqGetHDInfo
	if unmarshalData(ctx, &target) {

		if setting.WalletAddress == target.WalletAddress {
			SendMessageToSPServer(rspGetHDInfoData(), header.RspGetHDInfo)
		} else {
			transferSendMessageToClient(target.WalletAddress, spbf.MessageFromContext(ctx))
		}
	}
}

// RspGetHDInfo
func RspGetHDInfo(ctx context.Context, conn spbf.WriteCloser) {

	transferSendMessageToSPServer(spbf.MessageFromContext(ctx))
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
		stroeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqGetCapacity
func ReqGetCapacity(ctx context.Context, conn spbf.WriteCloser) {
	transferSendMessageToSPServer(spbf.MessageFromContext(ctx))
}

// RspGetCapacity
func RspGetCapacity(ctx context.Context, conn spbf.WriteCloser) {
	var target protos.RspGetCapacity
	if unmarshalData(ctx, &target) {
		if target.WalletAddress == setting.WalletAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				fmt.Println("action  successfully", target.Result.Msg)
			} else {
				fmt.Println("action  failed", target.Result.Msg)
			}
			putData(target.ReqId, HTTPGetCapacity, &target)
		} else {
			transferSendMessageToClient(target.WalletAddress, spbf.MessageFromContext(ctx))
		}
	}
}
