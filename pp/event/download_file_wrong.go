package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils"
)

func CheckAndSendRetryMessage(ctx context.Context, dTask *task.DownloadTask) {
	if !dTask.NeedRetry() {
		return
	}
	fileReqId, found := getFileReqIdFromContext(ctx)
	if !found {
		pp.DebugLog(ctx, "cannot find the original file request id")
		return
	}
	if f, ok := task.DownloadFileMap.Load(dTask.FileHash + fileReqId); ok {
		fInfo := f.(*protos.RspFileStorageInfo)
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqDownloadFileWrongData(fInfo, dTask), header.ReqDownloadFileWrong)
	}
}

// RspFileStorageInfo SP-PP , PP-P
func RspDownloadFileWrong(ctx context.Context, conn core.WriteCloser) {
	// PP check whether itself is the storage PP, if not transfer
	pp.Log(ctx, "get，RspDownloadFileWrong")
	var target protos.RspFileStorageInfo
	if err := VerifyMessage(ctx, header.RspDownloadFileWrong, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if requests.UnmarshalData(ctx, &target) {

		fileReqId, found := getFileReqIdFromContext(ctx)
		if !found {
			pp.DebugLog(ctx, "cannot find the original file request id")
			return
		}
		pp.DebugLog(ctx, "file hash", target.FileHash)
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			pp.Log(ctx, "download starts: ")
			dTask, ok := task.GetDownloadTask(target.FileHash, target.WalletAddress, fileReqId)
			if !ok {
				pp.DebugLog(ctx, "cannot find the download task")
				return
			}
			dTask.RefreshTask(&target)
			if target.IsVideoStream {
				return
			}
			if _, ok := task.DownloadSpeedOfProgress.Load(target.FileHash + fileReqId); !ok {
				pp.Log(ctx, "download has stopped")
				return
			}
			for _, slice := range target.SliceInfo {
				pp.DebugLog(ctx, "taskid ======= ", slice.TaskId)
				if file.CheckSliceExisting(target.FileHash, target.FileName, slice.SliceStorageInfo.SliceHash, target.SavePath, fileReqId) {
					pp.Log(ctx, "slice exist already,", slice.SliceStorageInfo.SliceHash)
					setDownloadSliceSuccess(ctx, slice.SliceStorageInfo.SliceHash, dTask)
					task.DownloadProgress(ctx, target.FileHash, fileReqId, slice.SliceOffset.SliceOffsetEnd-slice.SliceOffset.SliceOffsetStart)
				} else {
					pp.DebugLog(ctx, "request download data")
					req := requests.ReqDownloadSliceData(ctx, &target, slice)
					newCtx := createAndRegisterSliceReqId(ctx, fileReqId)
					SendReqDownloadSlice(newCtx, target.FileHash, slice, req, fileReqId)
				}
			}
			pp.DebugLog(ctx, "DownloadFileSlice(&target)", &target)
		} else {
			task.DeleteDownloadTask(target.FileHash, target.WalletAddress, task.LOCAL_REQID)
			pp.Log(ctx, "failed to download，", target.Result.Msg)
		}
	}
}
