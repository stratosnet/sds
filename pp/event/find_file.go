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

// FindMyFileList
func FindMyFileList(fileName, dir, reqID, keyword string, fileType int, isUp bool, w http.ResponseWriter) {
	if setting.CheckLogin() {
		sendMessage(client.PPConn, findMyFileListData(fileName, dir, reqID, keyword, protos.FileSortType(fileType), isUp), header.ReqFindMyFileList)
		stroeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqFindMyFileList ReqFindMyFileList
func ReqFindMyFileList(ctx context.Context, conn spbf.WriteCloser) {
	utils.DebugLog("+++++++++++++++++++++++++++++++++++++++++++++++++++")
	transferSendMessageToSPServer(spbf.MessageFromContext(ctx))
}

// RspFindMyFileList
func RspFindMyFileList(ctx context.Context, conn spbf.WriteCloser) {
	utils.DebugLog("get RspFindMyFileList")
	var target protos.RspFindMyFileList
	if unmarshalData(ctx, &target) {
		if target.WalletAddress == setting.WalletAddress {
			putData(target.ReqId, HTTPGetAllFile, &target)
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				if len(target.FileInfo) == 0 {
					fmt.Println("failed to get query file")
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
