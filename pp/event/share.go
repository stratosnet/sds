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
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/datamesh"
)

// GetAllShareLink GetShareLink
func GetAllShareLink(reqID, walletAddr string, page uint64, w http.ResponseWriter) {
	if setting.CheckLogin() {
		peers.SendMessageDirectToSPOrViaPP(requests.ReqShareLinkData(reqID, walletAddr, page), header.ReqShareLink)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// GetReqShareFile GetReqShareFile
func GetReqShareFile(reqID, fileHash, pathHash, walletAddr string, shareTime int64, isPrivate bool, w http.ResponseWriter) {
	if setting.CheckLogin() {
		peers.SendMessageDirectToSPOrViaPP(requests.ReqShareFileData(reqID, fileHash, pathHash, walletAddr, isPrivate, shareTime), header.ReqShareFile)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// DeleteShare DeleteShare
func DeleteShare(shareID, reqID, walletAddress string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		peers.SendMessageDirectToSPOrViaPP(requests.ReqDeleteShareData(reqID, shareID, walletAddress), header.ReqDeleteShare)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqShareLink
func ReqShareLink(ctx context.Context, conn core.WriteCloser) {
	// pp send to SP
	utils.DebugLog("ReqShareLinkReqShareLinkReqShareLinkReqShareLink")
	peers.TransferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspShareLink
func RspShareLink(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("RspShareLink(ctx context.Context, conn core.WriteCloser) {RspShareLink(ctx context.Context, conn core.WriteCloser) {")
	var target protos.RspShareLink
	rpcResult := &rpc.FileShareResult{}

	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	// serv the RPC user when the ReqId is not empty
	if target.ReqId != "" {
		defer file.SetFileShareResult(target.WalletAddress+target.ReqId, rpcResult)
	}

	if target.P2PAddress != setting.P2PAddress {
		peers.TransferSendMessageToPPServByP2pAddress(target.P2PAddress, core.MessageFromContext(ctx))
		rpcResult.Return = rpc.WRONG_PP_ADDRESS
		return
	}

	if target.Result.State == protos.ResultState_RES_SUCCESS {
		var fileInfos = make([]rpc.FileInfo, 0)
		for _, info := range target.ShareInfo {
			utils.Log("_______________________________")
			utils.Log("file_name:", info.Name)
			utils.Log("file_hash:", info.FileHash)
			utils.Log("file_size:", info.FileSize)
			utils.Log("link_time:", info.LinkTime)
			utils.Log("link_time_exp:", info.LinkTimeExp)
			utils.Log("ShareId:", info.ShareId)
			utils.Log("ShareLink:", info.ShareLink)
			fileInfos = append(fileInfos, rpc.FileInfo {
				FileHash: info.FileHash,
				FileSize: info.FileSize,
				FileName: info.Name,
				LinkTime: info.LinkTime,
				LinkTimeExp: info.LinkTimeExp,
				ShareId: info.ShareId,
				ShareLink: info.ShareLink,
			})
		}
		rpcResult.Return = rpc.SUCCESS
		rpcResult.FileInfo = fileInfos
		rpcResult.TotalNumber = target.TotalFileNumber
		rpcResult.PageId = target.PageId
	} else {
		utils.ErrorLog("all share failed:", target.Result.Msg)
		rpcResult.Return = rpc.INTERNAL_COMM_FAILURE
	}
	putData(target.ReqId, HTTPShareLink, &target)
	return
}

// ReqShareFile
func ReqShareFile(ctx context.Context, conn core.WriteCloser) {
	// pp send to SP
	peers.TransferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspShareFile
func RspShareFile(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspShareFile
	rpcResult := &rpc.FileShareResult{}

	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	if target.ReqId != "" {
		defer file.SetFileShareResult(target.WalletAddress+target.ReqId, rpcResult)
	}

	if target.P2PAddress != setting.P2PAddress {
		peers.TransferSendMessageToPPServByP2pAddress(target.P2PAddress, core.MessageFromContext(ctx))
		rpcResult.Return = rpc.WRONG_PP_ADDRESS
		return
	}

	if target.Result.State == protos.ResultState_RES_SUCCESS {
		utils.Log("ShareId", target.ShareId)
		utils.Log("ShareLink", target.ShareLink)
		utils.Log("SharePassword", target.SharePassword)
		rpcResult.Return = rpc.SUCCESS
		rpcResult.ShareId = target.ShareId
		rpcResult.ShareLink = target.ShareLink
	} else {
		utils.ErrorLog("share file failed:", target.Result.Msg)
		rpcResult.Return = rpc.INTERNAL_COMM_FAILURE
	}

	putData(target.ReqId, HTTPShareFile, &target)
	return
}

// ReqDeleteShare
func ReqDeleteShare(ctx context.Context, conn core.WriteCloser) {
	// pp send to SP
	peers.TransferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspDeleteShare
func RspDeleteShare(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspDeleteShare
	rpcResult := &rpc.FileShareResult{}

	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	if target.ReqId != "" {
		defer file.SetFileShareResult(target.WalletAddress+target.ReqId, rpcResult)
	}

	if target.P2PAddress != setting.P2PAddress {
		peers.TransferSendMessageToPPServByP2pAddress(target.P2PAddress, core.MessageFromContext(ctx))
		rpcResult.Return = rpc.WRONG_PP_ADDRESS
		return
	}

	if target.Result.State == protos.ResultState_RES_SUCCESS {
		utils.Log("cancel share success:", target.ShareId)
		rpcResult.Return = rpc.SUCCESS
	} else {
		utils.ErrorLog("cancel share failed:", target.Result.Msg)
		rpcResult.Return = rpc.GENERIC_ERR
	}
	putData(target.ReqId, HTTPDeleteShare, &target)
	return
}

// GetShareFile
func GetShareFile(keyword, sharePassword, saveAs, reqID, walletAddr string, w http.ResponseWriter) {
	utils.DebugLog("GetShareFile for file ", keyword)
	if setting.CheckLogin() {
		peers.SendMessageDirectToSPOrViaPP(requests.ReqGetShareFileData(keyword, sharePassword, saveAs, reqID, walletAddr), header.ReqGetShareFile)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqGetShareFile
func ReqGetShareFile(ctx context.Context, conn core.WriteCloser) {
	// pp send to SP
	utils.DebugLog("ReqGetShareFile: transferring message to SP server")
	peers.TransferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspGetShareFile
func RspGetShareFile(ctx context.Context, _ core.WriteCloser) {
	var target protos.RspGetShareFile
	rpcResult := &rpc.FileShareResult{}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	rpcRequested := target.ShareRequest.ReqId != task.LOCAL_REQID
	if rpcRequested {
		defer file.SetFileShareResult(target.ShareRequest.WalletAddress + target.ShareRequest.ReqId, rpcResult)
	}

	if target.ShareRequest.P2PAddress != setting.P2PAddress {
		peers.TransferSendMessageToPPServByP2pAddress(target.ShareRequest.P2PAddress, core.MessageFromContext(ctx))
		rpcResult.Return = rpc.WRONG_PP_ADDRESS
		return
	}

	utils.Log("get RspGetShareFile", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		rpcResult.Return = rpc.GENERIC_ERR
		return
	}

	utils.Log("FileInfo:", target.FileInfo)
	putData(target.ShareRequest.ReqId, HTTPGetShareFile, &target)

	for idx, fileInfo := range target.FileInfo {
		saveAs := ""
		if idx == 0 {
			saveAs = target.ShareRequest.SaveAs
		}
		filePath := datamesh.DataMashId{
			Owner: fileInfo.OwnerWalletAddress,
			Hash:  fileInfo.FileHash,
		}.String()

		var req *protos.ReqFileStorageInfo
		// notify rpc server starting file downloading
		if rpcRequested {
			f := rpc.FileInfo{FileHash: fileInfo.FileHash}
			rpcResult.Return = rpc.SHARED_DL_START
			rpcResult.FileInfo = append(rpcResult.FileInfo, f)
			file.SetFileShareResult(target.ShareRequest.WalletAddress + target.ShareRequest.ReqId, rpcResult)
			req, _ = requests.RequestDownloadFile(fileInfo.FileHash, fileInfo.OwnerWalletAddress, target.ShareRequest.ReqId,
				target.ShareRequest)
		} else {
			req = requests.ReqFileStorageInfoData(filePath, "", target.ShareRequest.ReqId, target.ShareRequest.WalletAddress,
				saveAs, false, target.ShareRequest)
		}
		peers.SendMessageDirectToSPOrViaPP(req, header.ReqFileStorageInfo)
	}
	return
}
