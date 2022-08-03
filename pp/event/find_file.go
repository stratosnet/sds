package event

import (
	"context"
	"net/http"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// FindMyFileList
func FindFileList(fileName string, walletAddr string, pageId uint64, reqID, keyword string, fileType int, isUp bool, w http.ResponseWriter) {
	if setting.CheckLogin() {
		peers.SendMessageDirectToSPOrViaPP(requests.FindFileListData(fileName, walletAddr, pageId, reqID, keyword, protos.FileSortType(fileType), isUp), header.ReqFindMyFileList)
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
	rpcResult := &rpc.FileListResult{}

	// fail to unmarshal data, not able to determine if and which RPC client this is from, let the client timeout
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	// serv the RPC user when the ReqId is not empty
	if target.ReqId != "" {
		defer file.SetFileListResult(target.WalletAddress+target.ReqId, rpcResult)
	}

	if target.P2PAddress != setting.P2PAddress {
		peers.TransferSendMessageToPPServByP2pAddress(target.P2PAddress, core.MessageFromContext(ctx))
		rpcResult.Return = rpc.WRONG_PP_ADDRESS
		return
	}

	putData(target.ReqId, HTTPGetAllFile, &target)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.ErrorLog(target.Result.Msg)
		rpcResult.Return = rpc.INTERNAL_COMM_FAILURE
		return
	}

	if len(target.FileInfo) == 0 {
		utils.Log("There are no files stored")
		rpcResult.Return = rpc.SUCCESS
		rpcResult.TotalNumber = target.TotalFileNumber
		rpcResult.PageId = target.PageId
		return
	}

	var fileInfos = make([]rpc.FileInfo, 0)
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
		fileInfos = append(fileInfos, rpc.FileInfo{
			FileHash: info.FileHash,
			FileSize: info.FileSize,
			FileName: info.FileName,
			CreateTime: info.CreateTime,
		})
	}

	utils.Log("===============================")
	utils.Logf("Total: %d  Page: %d", target.TotalFileNumber, target.PageId)

	rpcResult.Return = rpc.SUCCESS
	rpcResult.TotalNumber = target.TotalFileNumber
	rpcResult.PageId = target.PageId
	rpcResult.FileInfo = fileInfos

	return
}
