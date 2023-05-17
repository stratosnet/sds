package event

import (
	"context"

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

func GetAllShareLink(ctx context.Context, walletAddr string, page uint64) {
	if setting.CheckLogin() {
		p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, requests.ReqShareLinkData(walletAddr, p2pserver.GetP2pServer(ctx).GetP2PAddress(), page), header.ReqShareLink)
	}
}

func GetReqShareFile(ctx context.Context, fileHash, pathHash, walletAddr string, shareTime int64, isPrivate bool) {
	if setting.CheckLogin() {
		p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, requests.ReqShareFileData(fileHash, pathHash, walletAddr, p2pserver.GetP2pServer(ctx).GetP2PAddress(), isPrivate, shareTime), header.ReqShareFile)
	}
}

func DeleteShare(ctx context.Context, shareID, walletAddress string) {
	if setting.CheckLogin() {
		p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, requests.ReqDeleteShareData(shareID, walletAddress, p2pserver.GetP2pServer(ctx).GetP2PAddress()), header.ReqDeleteShare)
	}
}

func ReqShareLink(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqShareLink
	if err := VerifyMessage(ctx, header.ReqShareLink, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	// pp send to SP
	utils.DebugLog("ReqShareLinkReqShareLinkReqShareLinkReqShareLink")
	p2pserver.GetP2pServer(ctx).TransferSendMessageToSPServer(ctx, core.MessageFromContext(ctx))
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

func ReqShareFile(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqShareFile
	if err := VerifyMessage(ctx, header.ReqShareFile, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	// pp send to SP
	p2pserver.GetP2pServer(ctx).TransferSendMessageToSPServer(ctx, core.MessageFromContext(ctx))
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

func ReqDeleteShare(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqDeleteShare
	if err := VerifyMessage(ctx, header.ReqDeleteShare, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	// pp send to SP
	p2pserver.GetP2pServer(ctx).TransferSendMessageToSPServer(ctx, core.MessageFromContext(ctx))
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

func GetShareFile(ctx context.Context, keyword, sharePassword, saveAs, walletAddr string, walletPubkey []byte) {
	pp.DebugLog(ctx, "GetShareFile for file ", keyword)
	if setting.CheckLogin() {
		p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, requests.ReqGetShareFileData(keyword, sharePassword, saveAs, walletAddr, p2pserver.GetP2pServer(ctx).GetP2PAddress(), walletPubkey), header.ReqGetShareFile)
	}
}

func ReqGetShareFile(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqGetShareFile
	if err := VerifyMessage(ctx, header.ReqGetShareFile, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	// pp send to SP
	pp.DebugLog(ctx, "ReqGetShareFile: transferring message to SP server")
	p2pserver.GetP2pServer(ctx).TransferSendMessageToSPServer(ctx, core.MessageFromContext(ctx))
}

func RspGetShareFile(ctx context.Context, _ core.WriteCloser) {
	var target protos.RspGetShareFile
	if err := VerifyMessage(ctx, header.RspGetShareFile, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	rpcResult := &rpc.FileShareResult{}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	reqId := core.GetRemoteReqId(ctx)
	rpcRequested := reqId != task.LOCAL_REQID
	if rpcRequested {
		defer file.SetFileShareResult(target.ShareRequest.WalletAddress+reqId, rpcResult)
	}

	if target.ShareRequest.P2PAddress != p2pserver.GetP2pServer(ctx).GetP2PAddress() {
		p2pserver.GetP2pServer(ctx).TransferSendMessageToPPServByP2pAddress(ctx, target.ShareRequest.P2PAddress, core.MessageFromContext(ctx))
		rpcResult.Return = rpc.WRONG_PP_ADDRESS
		return
	}

	pp.Log(ctx, "get RspGetShareFile", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		rpcResult.Return = rpc.GENERIC_ERR
		return
	}

	pp.Log(ctx, "FileInfo:", target.FileInfo)

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
			file.SetFileShareResult(target.ShareRequest.WalletAddress+reqId, rpcResult)
			go func(fileInfo *protos.FileInfo) {
				if walletSign := file.GetSignatureFromRemote(fileInfo.FileHash); walletSign != nil {
					req = requests.RequestDownloadFile(ctx, fileInfo.FileHash, filePath, target.ShareRequest.WalletAddress, reqId, walletSign, target.ShareRequest.WalletPubkey, target.ShareRequest)
					p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, req, header.ReqFileStorageInfo)
				}
			}(fileInfo)

		} else {
			req = requests.ReqFileStorageInfoData(ctx, filePath, "", saveAs, setting.WalletAddress, setting.WalletPublicKey, false, target.ShareRequest)
			sig := utils.GetFileDownloadWalletSignMessage(fileInfo.FileHash, setting.WalletAddress, target.SequenceNumber)
			sign, err := types.BytesToAccPriveKey(setting.WalletPrivateKey).Sign([]byte(sig))
			if err != nil {
				return
			}
			req.WalletSign = sign
			p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, req, header.ReqFileStorageInfo)
		}
	}
}
