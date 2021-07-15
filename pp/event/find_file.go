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

// FindMyFileList
func FindMyFileList(fileName, dir, reqID, keyword string, fileType int, isUp bool, w http.ResponseWriter) {
	if setting.CheckLogin() {
		sendMessage(client.PPConn, findMyFileListData(fileName, dir, reqID, keyword, protos.FileSortType(fileType), isUp), header.ReqFindMyFileList)
		storeResponseWriter(reqID, w)
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
		if target.P2PAddress == setting.P2PAddress {
			putData(target.ReqId, HTTPGetAllFile, &target)
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				if len(target.FileInfo) == 0 {
					fmt.Println("There are no files stored")
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
