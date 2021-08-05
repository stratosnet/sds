package event

import (
	"context"
	"fmt"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"net/http"
)

// Invite
func Invite(code, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		sendMessage(client.PPConn, reqInviteData(code, reqID), header.ReqInvite)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqInvite
func ReqInvite(ctx context.Context, conn core.WriteCloser) {
	transferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspInvite
func RspInvite(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspInvite
	if unmarshalData(ctx, &target) {
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
			transferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}

// GetReward
func GetReward(reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		sendMessage(client.PPConn, reqGetRewardData(reqID), header.ReqGetReward)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqGetReward
func ReqGetReward(ctx context.Context, conn core.WriteCloser) {
	transferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspGetReward
func RspGetReward(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("RspGetReward>>>>>>>>>>>>>>>>>>>")
	var target protos.RspGetReward
	if unmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				fmt.Println("action  successfully", target.Result.Msg)
				fmt.Println("current capacity: ", target.CurrentCapacity)
			} else {
				fmt.Println("action  failed", target.Result.Msg)
			}
			putData(target.ReqId, HTTPReward, &target)
		} else {
			transferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}
