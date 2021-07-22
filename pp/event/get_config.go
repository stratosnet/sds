package event

import (
	"context"
	"fmt"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"net/http"
)

// GetMyConfig
func GetMyConfig(walletAddress, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		sendMessage(client.PPConn, reqGetMyConfig(walletAddress, reqID), header.ReqConfig)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqGetMyConfig ReqGetMyConfig
func ReqGetMyConfig(ctx context.Context, conn spbf.WriteCloser) {
	utils.DebugLog("+++++++++++++++++++++++++++++++++++++++++++++++++++")
	transferSendMessageToSPServer(spbf.MessageFromContext(ctx))
}

// RspGetMyConfig
func RspGetMyConfig(ctx context.Context, conn spbf.WriteCloser) {
	utils.DebugLog("get RspConfig")
	var target protos.RspConfig
	if unmarshalData(ctx, &target) {
		if target.WalletAddress == setting.WalletAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				fmt.Println("action  successfully", target.Result.Msg)
			} else {
				fmt.Println("action  failed", target.Result.Msg)
			}
			putData(target.ReqId, HTTPGetConfig, &target)
		} else {
			transferSendMessageToClient(target.WalletAddress, spbf.MessageFromContext(ctx))
		}
	}
}
