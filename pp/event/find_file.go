package event

import (
	"context"
	"net/http"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// FindMyFileList
func FindMyFileList(fileName, dir, reqID, keyword string, fileType int, isUp bool, w http.ResponseWriter) {
	if setting.CheckLogin() {
		peers.SendMessageDirectToSPOrViaPP(requests.FindMyFileListData(fileName, dir, reqID, keyword, protos.FileSortType(fileType), isUp), header.ReqFindMyFileList)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqFindMyFileList ReqFindMyFileList
func ReqFindMyFileList(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("+++++++++++++++++++++++++++++++++++++++++++++++++++")
	peers.TransferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspFindMyFileList
func RspFindMyFileList(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get RspFindMyFileList")
	var target protos.RspFindMyFileList
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	if target.P2PAddress != setting.P2PAddress {
		peers.TransferSendMessageToPPServByP2pAddress(target.P2PAddress, core.MessageFromContext(ctx))
		return
	}

	putData(target.ReqId, HTTPGetAllFile, &target)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.ErrorLog(target.Result.Msg)
		return
	}

	if len(target.FileInfo) == 0 {
		utils.Log("There are no files stored")
		return
	}
	for _, info := range target.FileInfo {
		utils.Log("_______________________________")
		if info.IsDirectory {
			utils.Log("Directory name:", info.FileName)
			utils.Log("Directory hash:", info.FileHash)
		} else {
			utils.Log("File name:", info.FileName)
			utils.Log("File hash:", info.FileHash)
		}
		utils.Log("CreateTime :", info.CreateTime)
	}
}
