package event

import (
	"context"
	"sync"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/datamesh"
	"github.com/stratosnet/sds/utils/types"
)

var (
	uploadRequestMap   = &sync.Map{}
	downloadRequestMap = &sync.Map{}
)

// GetWalletOz queries current ozone balance
func GetWalletOz(ctx context.Context, walletAddr, reqId string) error {
	pp.Logf(ctx, "Querying current ozone balance of the wallet: %v", walletAddr)
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqGetWalletOzData(walletAddr, reqId), header.ReqGetWalletOz)
	return nil
}

func ReqGetWalletOzForUpload(ctx context.Context, walletAddr, reqId string, uploadReq *protos.ReqUploadFile) error {
	uploadRequestMap.Store(core.GetReqIdFromContext(ctx), uploadReq)
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqGetWalletOzData(walletAddr, reqId), header.ReqGetWalletOz)
	return nil
}

func ReqGetWalletOzForDownload(ctx context.Context, walletAddr, reqId string, downloadReq *protos.ReqFileStorageInfo) error {
	downloadRequestMap.Store(core.GetReqIdFromContext(ctx), downloadReq)
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqGetWalletOzData(walletAddr, reqId), header.ReqGetWalletOz)
	return nil
}

func RspGetWalletOz(ctx context.Context, conn core.WriteCloser) {
	pp.DebugLog(ctx, "get GetWalletOz RSP")
	var target protos.RspGetWalletOz
	if err := VerifyMessage(ctx, header.RspGetWalletOz, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		pp.DebugLog(ctx, "Cannot unmarshal ozone balance data")
		return
	}
	rpcResult := &rpc.GetOzoneResult{}
	reqId := core.GetRemoteReqId(ctx)
	if reqId != "" {
		defer file.SetQueryOzoneResult(target.WalletAddress+reqId, rpcResult)
	}
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		pp.Logf(ctx, "failed to get ozone balance: %v", target.Result.Msg)
		rpcResult.Return = rpc.INTERNAL_COMM_FAILURE
		return
	}

	if reqMsg, loaded := uploadRequestMap.LoadAndDelete(requests.GetReqIdFromMessage(ctx)); loaded {
		rmsg := reqMsg.(*protos.ReqUploadFile)
		walletString := utils.GetFileUploadWalletSignMessage(rmsg.FileInfo.FileHash, setting.WalletAddress, target.SequenceNumber, rmsg.ReqTime)
		wsign, err := types.BytesToAccPriveKey(setting.WalletPrivateKey).Sign([]byte(walletString))
		if err != nil {
			return
		}
		rmsg.Signature.Signature = wsign
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, rmsg, header.ReqUploadFile)
		return
	}

	if reqMsg, loaded := downloadRequestMap.LoadAndDelete(requests.GetReqIdFromMessage(ctx)); loaded {
		rmsg := reqMsg.(*protos.ReqFileStorageInfo)
		_, _, fileHash, _, err := datamesh.ParseFileHandle(rmsg.FileIndexes.FilePath)
		if err != nil {
			return
		}
		walletString := utils.GetFileDownloadWalletSignMessage(fileHash, setting.WalletAddress, target.SequenceNumber, rmsg.ReqTime)
		wsign, err := types.BytesToAccPriveKey(setting.WalletPrivateKey).Sign([]byte(walletString))
		if err != nil {
			return
		}
		rmsg.Signature.Signature = wsign
		p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, rmsg, header.ReqFileStorageInfo)
		return
	}
	pp.Logf(ctx, "get GetWalletOz RSP, the current ozone balance of %v = %v, %s, %v", target.GetWalletAddress(), target.GetWalletOz(), target.SequenceNumber, reqId)
	rpcResult.Return = rpc.SUCCESS
	rpcResult.Ozone = target.WalletOz
	rpcResult.SequenceNumber = target.SequenceNumber
}
