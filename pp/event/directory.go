package event

import (
	"context"
	"fmt"
	"github.com/qsnetwork/qsds/framework/spbf"
	"github.com/qsnetwork/qsds/msg/header"
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/pp/client"
	"github.com/qsnetwork/qsds/pp/setting"
	"github.com/qsnetwork/qsds/utils"
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
		if target.WalletAddress == setting.WalletAddress {
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
			transferSendMessageToClient(target.WalletAddress, spbf.MessageFromContext(ctx))
		}

	}
}
