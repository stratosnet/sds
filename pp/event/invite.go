package event

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// Invite
func Invite(code, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		peers.SendMessage(client.PPConn, requests.ReqInviteData(code, reqID), header.ReqInvite)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqInvite
func ReqInvite(ctx context.Context, conn core.WriteCloser) {
	peers.TransferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspInvite
func RspInvite(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspInvite
	if requests.UnmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				fmt.Println("action  successfully", target.Result.Msg)
				fmt.Println("added capacity: ", target.CapacityDelta)
				fmt.Println("total capacity: ", target.CurrentCapacity)
			} else {
				fmt.Println("action  failed", target.Result.Msg)
			}
			putData(target.ReqId, HTTPInvite, &target)
		} else {
			peers.TransferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}

// GetReward
func GetReward(reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		peers.SendMessage(client.PPConn, requests.ReqGetRewardData(reqID), header.ReqGetReward)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqGetReward
func ReqGetReward(ctx context.Context, conn core.WriteCloser) {
	peers.TransferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspGetReward
func RspGetReward(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("RspGetReward>>>>>>>>>>>>>>>>>>>")
	var target protos.RspGetReward
	if requests.UnmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				fmt.Println("action  successfully", target.Result.Msg)
				fmt.Println("current capacity: ", target.CurrentCapacity)
			} else {
				fmt.Println("action  failed", target.Result.Msg)
			}
			putData(target.ReqId, HTTPReward, &target)
		} else {
			peers.TransferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}
