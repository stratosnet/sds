package event

import (
	"context"
	"sync"
	"time"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/crypto"
	"github.com/stratosnet/sds/framework/msg/header"
	fwutils "github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/metrics"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/sds-msg/protos"
)

var (

	// key: fileHash + fileReqId; value: sdm (already got translated from share link)
	sdmMap = &sync.Map{}
)

func GetAllShareLink(ctx context.Context, walletAddr string, page uint64, walletPubkey, wsign []byte, reqTime int64) {
	if setting.CheckLogin() {
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(
			ctx,
			requests.ReqShareLinkData(
				walletAddr, p2pserver.GetP2pServer(ctx).GetP2PAddress().String(),
				page, walletPubkey, wsign, reqTime,
			),
			header.ReqShareLink,
		)
	}
}

func ReqShareFile(ctx context.Context, fileHash, pathHash, walletAddr string, shareTime int64, isPrivate bool,
	walletPubkey, wsign []byte, reqTime int64, ipfsCid string) {
	if setting.CheckLogin() {
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(
			ctx,
			requests.ReqShareFileData(
				fileHash, pathHash, walletAddr, p2pserver.GetP2pServer(ctx).GetP2PAddress().String(),
				isPrivate, shareTime, walletPubkey, wsign, reqTime, ipfsCid,
			),
			header.ReqShareFile,
		)
	}
}

func DeleteShare(ctx context.Context, shareID, walletAddress string, walletPubkey, wsign []byte, reqTime int64) {
	if setting.CheckLogin() {
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(
			ctx,
			requests.ReqDeleteShareData(
				shareID, walletAddress, p2pserver.GetP2pServer(ctx).GetP2PAddress().String(),
				walletPubkey, wsign, reqTime,
			),
			header.ReqDeleteShare,
		)
	}
}

func RspShareLink(ctx context.Context, conn core.WriteCloser) {
	pp.DebugLog(ctx, "RspShareLink(ctx context.Context, conn core.WriteCloser) {RspShareLink(ctx context.Context, conn core.WriteCloser) {")
	var target protos.RspShareLink
	if err := VerifyMessage(ctx, header.RspShareLink, &target); err != nil {
		fwutils.ErrorLog("failed verifying the message, ", err.Error())
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

	if target.P2PAddress != p2pserver.GetP2pServer(ctx).GetP2PAddress().String() {
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
		fwutils.ErrorLog("failed verifying the message, ", err.Error())
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

	if target.P2PAddress != p2pserver.GetP2pServer(ctx).GetP2PAddress().String() {
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
		rpcResult.Return = target.Result.Msg
	}
}

func RspDeleteShare(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspDeleteShare
	if err := VerifyMessage(ctx, header.RspDeleteShare, &target); err != nil {
		fwutils.ErrorLog("failed verifying the message, ", err.Error())
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

	if target.P2PAddress != p2pserver.GetP2pServer(ctx).GetP2PAddress().String() {
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

func GetShareFile(ctx context.Context, keyword, sharePassword, saveAs, walletAddr string, walletPubkey []byte, wsign []byte, reqTime int64) {
	pp.DebugLog(ctx, "GetShareFile for file ", keyword)
	if setting.CheckLogin() {
		req := requests.ReqGetShareFileData(
			keyword, sharePassword, saveAs, walletAddr, p2pserver.GetP2pServer(ctx).GetP2PAddress().String(),
			walletPubkey, wsign, reqTime,
		)
		_ = ReqGetWalletOzForGetShareFile(ctx, setting.WalletAddress, task.LOCAL_REQID, req)
	}
}

func RspGetShareFile(ctx context.Context, _ core.WriteCloser) {
	var target protos.RspFileStorageInfo
	if err := VerifyMessage(ctx, header.RspGetShareFile, &target); err != nil {
		fwutils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}

	// SPAM check
	if time.Now().Unix()-target.TimeStamp > setting.SpamThresholdSpSignLatency {
		pp.ErrorLog(ctx, "sp's get shared file response was expired")
		return
	}

	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	reqId := core.GetRemoteReqId(ctx)
	rpcResult := &rpc.FileShareResult{}
	key := target.KeyWord + reqId
	defer file.SetFileShareResult(key, rpcResult)

	if target.Result.State == protos.ResultState_RES_FAIL {
		task.DownloadResult(ctx, target.FileHash, false, "failed ReqGetSharedFile, "+target.Result.Msg)
		rpcResult.Return = target.Result.Msg
		return
	}
	metrics.DownloadPerformanceLogNow(target.FileHash + ":RCV_STORAGE_INFO_SP:")

	newTarget := &target
	newTarget.ReqId = reqId
	pp.DebugLog(ctx, "file hash, reqid:", target.FileHash, reqId)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		file.SetDownloadSliceResult(target.FileHash, false)
		return
	}

	task.CleanDownloadFileAndConnMap(ctx, target.FileHash, reqId)
	task.DownloadFileMap.Store(target.FileHash+reqId, newTarget)
	task.AddDownloadTask(newTarget)
	file.StartLocalDownload(target.FileHash)
	var slice *protos.DownloadSliceInfo
	var slices []file.DownloadSlice
	for _, slice = range target.SliceInfo {
		s := file.DownloadSlice{
			SliceHash:       slice.SliceStorageInfo.SliceHash,
			SliceFileOffset: slice.SliceOffset.SliceOffsetStart,
			SliceSize:       slice.SliceStorageInfo.SliceSize,
		}
		slices = append(slices, s)
	}

	file.SetDownloadFileInfo(target.FileHash, file.DownloadFile{
		FileHash: target.FileHash,
		FileSize: target.FileSize,
		FileName: target.FileName,
		Slices:   slices,
	})

	f := rpc.FileInfo{FileHash: target.FileHash, FileName: target.FileName, FileSize: target.FileSize}
	rpcResult.Return = rpc.SHARED_DL_START
	rpcResult.FileInfo = append(rpcResult.FileInfo, f)

	if crypto.IsVideoStream(target.FileHash) {
		return
	}
	DownloadFileSlices(ctx, newTarget, reqId)
}

func GetFilePath(key string) string {
	filePath, ok := sdmMap.Load(key)
	if !ok {
		fwutils.DebugLog("FAILED!")
		return ""
	}

	return filePath.(string)
}
