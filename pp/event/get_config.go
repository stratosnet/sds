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

// GetMyConfig
func GetMyConfig(p2pAddress, walletAddress, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		peers.SendMessage(client.PPConn, requests.ReqGetMyConfig(p2pAddress, walletAddress, reqID), header.ReqConfig)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqGetMyConfig ReqGetMyConfig
func ReqGetMyConfig(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("+++++++++++++++++++++++++++++++++++++++++++++++++++")
	peers.TransferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspGetMyConfig
func RspGetMyConfig(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get RspConfig")
	var target protos.RspConfig
	if requests.UnmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				fmt.Println("action  successfully", target.Result.Msg)
			} else {
				fmt.Println("action  failed", target.Result.Msg)
			}
			putData(target.ReqId, HTTPGetConfig, &target)
		} else {
			peers.TransferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}
