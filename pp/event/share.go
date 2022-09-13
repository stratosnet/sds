package event

import (
	"context"
	"github.com/stratosnet/sds/utils/types"
	"net/http"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/datamesh"
)

// GetAllShareLink GetShareLink
func GetAllShareLink(ctx context.Context, reqID, walletAddr string, page uint64, w http.ResponseWriter) {
	if setting.CheckLogin() {
		peers.SendMessageDirectToSPOrViaPP(ctx, requests.ReqShareLinkData(reqID, walletAddr, page), header.ReqShareLink)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// GetReqShareFile GetReqShareFile
func GetReqShareFile(ctx context.Context, reqID, fileHash, pathHash, walletAddr string, shareTime int64, isPrivate bool, w http.ResponseWriter) {
	if setting.CheckLogin() {
		peers.SendMessageDirectToSPOrViaPP(ctx, requests.ReqShareFileData(reqID, fileHash, pathHash, walletAddr, isPrivate, shareTime), header.ReqShareFile)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// DeleteShare DeleteShare
func DeleteShare(ctx context.Context, shareID, reqID, walletAddress string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		peers.SendMessageDirectToSPOrViaPP(ctx, requests.ReqDeleteShareData(reqID, shareID, walletAddress), header.ReqDeleteShare)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqShareLink
func ReqShareLink(ctx context.Context, conn core.WriteCloser) {
	// pp send to SP
	utils.DebugLog("ReqShareLinkReqShareLinkReqShareLinkReqShareLink")
	peers.TransferSendMessageToSPServer(ctx, core.MessageFromContext(ctx))
}

// RspShareLink
func RspShareLink(ctx context.Context, conn core.WriteCloser) {
	pp.DebugLog(ctx, "RspShareLink(ctx context.Context, conn core.WriteCloser) {RspShareLink(ctx context.Context, conn core.WriteCloser) {")
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
		peers.TransferSendMessageToPPServByP2pAddress(ctx, target.P2PAddress, core.MessageFromContext(ctx))
		rpcResult.Return = rpc.WRONG_PP_ADDRESS
		return
	}

	if target.Result.State == protos.ResultState_RES_SUCCESS {
		var fileInfos = make([]rpc.FileInfo, 0)
		for _, info := range target.ShareInfo {
			pp.Log(ctx, "_______________________________")
			pp.Log(ctx, "file_name:", info.Name)
			pp.Log(ctx, "file_hash:", info.FileHash)
			pp.Log(ctx, "file_size:", info.FileSize)
			pp.Log(ctx, "link_time:", info.LinkTime)
			pp.Log(ctx, "link_time_exp:", info.LinkTimeExp)
			pp.Log(ctx, "ShareId:", info.ShareId)
			pp.Log(ctx, "ShareLink:", info.ShareLink)
			fileInfos = append(fileInfos, rpc.FileInfo{
				FileHash:    info.FileHash,
				FileSize:    info.FileSize,
				FileName:    info.Name,
				LinkTime:    info.LinkTime,
				LinkTimeExp: info.LinkTimeExp,
				ShareId:     info.ShareId,
				ShareLink:   info.ShareLink,
			})
		}
		rpcResult.Return = rpc.SUCCESS
		rpcResult.FileInfo = fileInfos
		rpcResult.TotalNumber = target.TotalFileNumber
		rpcResult.PageId = target.PageId
	} else {
		pp.ErrorLog(ctx, "all share failed:", target.Result.Msg)
		rpcResult.Return = rpc.INTERNAL_COMM_FAILURE
	}
	putData(target.ReqId, HTTPShareLink, &target)
	return
}

// ReqShareFile
func ReqShareFile(ctx context.Context, conn core.WriteCloser) {
	// pp send to SP
	peers.TransferSendMessageToSPServer(ctx, core.MessageFromContext(ctx))
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
		peers.TransferSendMessageToPPServByP2pAddress(ctx, target.P2PAddress, core.MessageFromContext(ctx))
		rpcResult.Return = rpc.WRONG_PP_ADDRESS
		return
	}

	if target.Result.State == protos.ResultState_RES_SUCCESS {
		pp.Log(ctx, "ShareId", target.ShareId)
		pp.Log(ctx, "ShareLink", target.ShareLink)
		pp.Log(ctx, "SharePassword", target.SharePassword)
		rpcResult.Return = rpc.SUCCESS
		rpcResult.ShareId = target.ShareId
		rpcResult.ShareLink = target.ShareLink
	} else {
		pp.ErrorLog(ctx, "share file failed:", target.Result.Msg)
		rpcResult.Return = rpc.INTERNAL_COMM_FAILURE
	}

	putData(target.ReqId, HTTPShareFile, &target)
	return
}

// ReqDeleteShare
func ReqDeleteShare(ctx context.Context, conn core.WriteCloser) {
	// pp send to SP
	peers.TransferSendMessageToSPServer(ctx, core.MessageFromContext(ctx))
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
		peers.TransferSendMessageToPPServByP2pAddress(ctx, target.P2PAddress, core.MessageFromContext(ctx))
		rpcResult.Return = rpc.WRONG_PP_ADDRESS
		return
	}

	if target.Result.State == protos.ResultState_RES_SUCCESS {
		pp.Log(ctx, "cancel share success:", target.ShareId)
		rpcResult.Return = rpc.SUCCESS
	} else {
		pp.ErrorLog(ctx, "cancel share failed:", target.Result.Msg)
		rpcResult.Return = rpc.GENERIC_ERR
	}
	putData(target.ReqId, HTTPDeleteShare, &target)
	return
}

// GetShareFile
func GetShareFile(ctx context.Context, keyword, sharePassword, saveAs, reqID, walletAddr string, walletPubkey, walletSign []byte, w http.ResponseWriter) {
	pp.DebugLog(ctx, "GetShareFile for file ", keyword)
	if setting.CheckLogin() {
		peers.SendMessageDirectToSPOrViaPP(ctx, requests.ReqGetShareFileData(keyword, sharePassword, saveAs, reqID, walletAddr, walletPubkey, walletSign), header.ReqGetShareFile)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqGetShareFile
func ReqGetShareFile(ctx context.Context, conn core.WriteCloser) {
	// pp send to SP
	pp.DebugLog(ctx, "ReqGetShareFile: transferring message to SP server")
	peers.TransferSendMessageToSPServer(ctx, core.MessageFromContext(ctx))
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
		defer file.SetFileShareResult(target.ShareRequest.WalletAddress+target.ShareRequest.ReqId, rpcResult)
	}

	if target.ShareRequest.P2PAddress != setting.P2PAddress {
		peers.TransferSendMessageToPPServByP2pAddress(ctx, target.ShareRequest.P2PAddress, core.MessageFromContext(ctx))
		rpcResult.Return = rpc.WRONG_PP_ADDRESS
		return
	}

	pp.Log(ctx, "get RspGetShareFile", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		rpcResult.Return = rpc.GENERIC_ERR
		return
	}

	pp.Log(ctx, "FileInfo:", target.FileInfo)
	putData(target.ShareRequest.ReqId, HTTPGetShareFile, &target)

	for idx, fileInfo := range target.FileInfo {
		saveAs := ""
		if idx == 0 {
			saveAs = target.ShareRequest.SaveAs
		}
		filePath := datamesh.DataMeshId{
			Owner: fileInfo.OwnerWalletAddress,
			Hash:  fileInfo.FileHash,
		}.String()

		var req *protos.ReqFileStorageInfo
		// notify rpc server starting file downloading
		if rpcRequested {
			f := rpc.FileInfo{FileHash: fileInfo.FileHash}
			rpcResult.Return = rpc.SHARED_DL_START
			rpcResult.FileInfo = append(rpcResult.FileInfo, f)
			file.SetFileShareResult(target.ShareRequest.WalletAddress+target.ShareRequest.ReqId, rpcResult)
			req, _ = requests.RequestDownloadFile(fileInfo.FileHash, filePath, target.ShareRequest.WalletAddress, target.ShareRequest.ReqId, target.ShareRequest.WalletSign, target.ShareRequest.WalletPubkey, target.ShareRequest)
		} else {
			sig := utils.GetFileDownloadShareWalletSignMessage(fileInfo.FileHash, setting.WalletAddress)
			sign, err := types.BytesToAccPriveKey(setting.WalletPrivateKey).Sign([]byte(sig))
			if err != nil {
				return
			}
			req = requests.ReqFileStorageInfoData(filePath, "", target.ShareRequest.ReqId, saveAs, setting.WalletAddress, sign, setting.WalletPublicKey, false, target.ShareRequest)
		}
		peers.SendMessageDirectToSPOrViaPP(ctx, req, header.ReqFileStorageInfo)
	}
	return
}
