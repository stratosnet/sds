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

// FindDirectory
func FindDirectory(reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		sendMessage(client.PPConn, findDirectoryData(reqID), header.ReqFindDirectory)
		stroeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqFindDirectory ReqFindDirectory
func ReqFindDirectory(ctx context.Context, conn spbf.WriteCloser) {
	transferSendMessageToSPServer(spbf.MessageFromContext(ctx))
}

// RspFindDirectory
func RspFindDirectory(ctx context.Context, conn spbf.WriteCloser) {
	utils.DebugLog("RspFindDirectory")
	var target protos.RspFindDirectory
	if unmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			putData(target.ReqId, HTTPGetAllDirectory, &target)
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				if len(target.FileInfo) == 0 {
					fmt.Println("no directory")
					return
				}
				for _, info := range target.FileInfo {

					fmt.Println("_______________________________")
					if info.IsDirectory {
						fmt.Println("Directory:", info.FileName)
					} else {
						fmt.Println("name:", info.FileName)
						fmt.Println("hash:", info.FileHash)
					}
					fmt.Println("CreateTime :", info.CreateTime)

				}
			} else {
				utils.ErrorLog(target.Result.Msg)
				fmt.Println(target.Result.Msg)
			}
		} else {
			transferSendMessageToClient(target.P2PAddress, spbf.MessageFromContext(ctx))
		}

	}
}
