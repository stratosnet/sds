package event

import (
	"context"
	"strings"
	"sync"

	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils/types"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/datamesh"
)

var (

	// key: fileHash + fileReqId; value: sdm (already got translated from share link)
	sdmMap = &sync.Map{}
)

func GetAllShareLink(ctx context.Context, walletAddr string, page uint64, walletPubkey, wsign []byte, reqTime int64) {
	if setting.CheckLogin() {
		p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, requests.ReqShareLinkData(walletAddr, p2pserver.GetP2pServer(ctx).GetP2PAddress(), page, walletPubkey, wsign, reqTime), header.ReqShareLink)
	}
}

func GetReqShareFile(ctx context.Context, fileHash, pathHash, walletAddr string, shareTime int64, isPrivate bool,
	walletPubkey, wsign []byte, reqTime int64) {
	if setting.CheckLogin() {
		p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx,
			requests.ReqShareFileData(fileHash, pathHash, walletAddr,
				p2pserver.GetP2pServer(ctx).GetP2PAddress(), isPrivate, shareTime,
				walletPubkey, wsign, reqTime), header.ReqShareFile)
	}
}

func DeleteShare(ctx context.Context, shareID, walletAddress string, walletPubkey, wsign []byte, reqTime int64) {
	if setting.CheckLogin() {
		p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, requests.ReqDeleteShareData(shareID, walletAddress, p2pserver.GetP2pServer(ctx).GetP2PAddress(), walletPubkey, wsign, reqTime), header.ReqDeleteShare)
	}
}

func RspShareLink(ctx context.Context, conn core.WriteCloser) {
	pp.DebugLog(ctx, "RspShareLink(ctx context.Context, conn core.WriteCloser) {RspShareLink(ctx context.Context, conn core.WriteCloser) {")
	var target protos.RspShareLink
	if err := VerifyMessage(ctx, header.RspShareLink, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	rpcResult := &rpc.FileShareResult{}

	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	// serv the RPC user when the ReqId is not empty
	reqId := core.GetRemoteReqId(ctx)
	if reqId != "" {
		defer file.SetFileShareResult(target.WalletAddress+reqId, rpcResult)
	}

	if target.P2PAddress != p2pserver.GetP2pServer(ctx).GetP2PAddress() {
		p2pserver.GetP2pServer(ctx).TransferSendMessageToPPServByP2pAddress(ctx, target.P2PAddress, core.MessageFromContext(ctx))
		rpcResult.Return = rpc.WRONG_PP_ADDRESS
		return
	}

	if target.Result.State == protos.ResultState_RES_SUCCESS {
		var fileInfos = make([]rpc.FileInfo, 0)

		if len(target.ShareInfo) == 0 {
			pp.Log(ctx, "no shared file found")
		} else {
			for _, info := range target.ShareInfo {
				pp.Log(ctx, "_______________________________")
				pp.Log(ctx, "file_name:", info.Name)
				pp.Log(ctx, "file_hash:", info.FileHash)
				pp.Log(ctx, "file_size:", info.FileSize)

				pp.Log(ctx, "share_creation_time:", info.CreationTime)
				pp.Log(ctx, "share_exp_time:", info.ExpTime)
				pp.Log(ctx, "ShareId:", info.ShareId)
				pp.Log(ctx, "ShareLink:", info.ShareLink)
				fileInfos = append(fileInfos, rpc.FileInfo{
					FileHash:    info.FileHash,
					FileSize:    info.FileSize,
					FileName:    info.Name,
					LinkTime:    info.CreationTime,
					LinkTimeExp: info.ExpTime,
					ShareId:     info.ShareId,
					ShareLink:   info.ShareLink,
				})
			}
		}
		rpcResult.Return = rpc.SUCCESS
		rpcResult.FileInfo = fileInfos
		rpcResult.TotalNumber = target.TotalFileNumber
		rpcResult.PageId = target.PageId
	} else {
		pp.ErrorLog(ctx, "all share failed:", target.Result.Msg)
		rpcResult.Return = rpc.INTERNAL_COMM_FAILURE
	}
}

func RspShareFile(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspShareFile
	if err := VerifyMessage(ctx, header.RspShareFile, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	rpcResult := &rpc.FileShareResult{}

	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	reqId := core.GetRemoteReqId(ctx)
	if reqId != "" {
		defer file.SetFileShareResult(target.WalletAddress+reqId, rpcResult)
	}

	if target.P2PAddress != p2pserver.GetP2pServer(ctx).GetP2PAddress() {
		p2pserver.GetP2pServer(ctx).TransferSendMessageToPPServByP2pAddress(ctx, target.P2PAddress, core.MessageFromContext(ctx))
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
}

func RspDeleteShare(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspDeleteShare
	if err := VerifyMessage(ctx, header.RspDeleteShare, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	rpcResult := &rpc.FileShareResult{}

	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	reqId := core.GetRemoteReqId(ctx)
	if reqId != "" {
		defer file.SetFileShareResult(target.WalletAddress+reqId, rpcResult)
	}

	if target.P2PAddress != p2pserver.GetP2pServer(ctx).GetP2PAddress() {
		p2pserver.GetP2pServer(ctx).TransferSendMessageToPPServByP2pAddress(ctx, target.P2PAddress, core.MessageFromContext(ctx))
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
}

func GetShareFile(ctx context.Context, keyword, sharePassword, saveAs, walletAddr string, walletPubkey []byte,
	isVideoStream bool, wsign []byte, reqTime int64) {
	pp.DebugLog(ctx, "GetShareFile for file ", keyword)
	if setting.CheckLogin() {
		p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, requests.ReqGetShareFileData(keyword, sharePassword, saveAs, walletAddr, p2pserver.GetP2pServer(ctx).GetP2PAddress(), walletPubkey, wsign, isVideoStream, reqTime), header.ReqGetShareFile)
	}
}

func RspGetShareFile(ctx context.Context, _ core.WriteCloser) {
	var target protos.RspGetShareFile
	if err := VerifyMessage(ctx, header.RspGetShareFile, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	reqId := core.GetRemoteReqId(ctx)
	rpcRequested := !strings.HasPrefix(reqId, task.LOCAL_REQID)
	rpcResult := &rpc.FileShareResult{}
	if rpcRequested {
		defer file.SetFileShareResult(target.ShareRequest.Signature.Address+reqId, rpcResult)
	}

	if target.Result.State != protos.ResultState_RES_SUCCESS {
		rpcResult.Return = rpc.GENERIC_ERR
		return
	}

	if target.ShareRequest == nil {
		utils.ErrorLog("got empty ShareRequest from sp")
		return
	}

	if target.ShareRequest.P2PAddress != p2pserver.GetP2pServer(ctx).GetP2PAddress() {
		p2pserver.GetP2pServer(ctx).TransferSendMessageToPPServByP2pAddress(ctx, target.ShareRequest.P2PAddress, core.MessageFromContext(ctx))
		rpcResult.Return = rpc.WRONG_PP_ADDRESS
		return
	}

	utils.Log("get RspGetShareFile", target.Result.State, target.Result.Msg)
	utils.Log("FileInfo:", target.FileInfo)

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
			rpcResult.SequenceNumber = target.SequenceNumber
			sdmMap.Store(fileInfo.FileHash+reqId, filePath)
			file.SetFileShareResult(target.ShareRequest.Signature.Address+reqId, rpcResult)
		} else {
			if task.CheckDownloadTask(fileInfo.FileHash, setting.WalletAddress, task.LOCAL_REQID) {
				pp.DebugLog(ctx, "* This file is being downloaded, please wait and try later\n")
				return
			}
			req = requests.ReqFileStorageInfoData(ctx, filePath, "", saveAs, setting.WalletAddress, setting.WalletPublicKey, nil, target.ShareRequest, target.ShareRequest.ReqTime)
			sigMsg := utils.GetFileDownloadWalletSignMessage(fileInfo.FileHash, setting.WalletAddress, target.SequenceNumber, target.ShareRequest.ReqTime)
			sign, err := types.BytesToAccPriveKey(setting.WalletPrivateKey).Sign([]byte(sigMsg))
			if err != nil {
				return
			}
			req.Signature.Signature = sign
			p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, req, header.ReqFileStorageInfo)
		}
	}
}

func GetFilePath(key string) string {
	filePath, ok := sdmMap.Load(key)
	if !ok {
		utils.DebugLog("FAILED!")
		return ""
	}

	return filePath.(string)
}
