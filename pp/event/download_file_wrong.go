package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/task"
)

func CheckAndSendRetryMessage(ctx context.Context, dTask *task.DownloadTask) {
	if !dTask.NeedRetry() {
		return
	}
	if f, ok := task.DownloadFileMap.Load(dTask.FileHash + task.LOCAL_REQID); ok {
		fInfo := f.(*protos.RspFileStorageInfo)
		peers.SendMessageToSPServer(ctx, requests.ReqDownloadFileWrongData(fInfo, dTask), header.ReqDownloadFileWrong)
	}
}

// RspFileStorageInfo SP-PP , PP-P
func RspDownloadFileWrong(ctx context.Context, conn core.WriteCloser) {
	// PP check whether itself is the storage PP, if not transfer
	pp.Log(ctx, "get，RspDownloadFileWrong")
	var target protos.RspFileStorageInfo
	if requests.UnmarshalData(ctx, &target) {
		pp.DebugLog(ctx, "file hash", target.FileHash)
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			pp.Log(ctx, "download starts: ")
			dTask, ok := task.GetDownloadTask(target.FileHash, target.WalletAddress, target.ReqId)
			if !ok {
				pp.DebugLog(ctx, "cannot find the download task")
				return
			}
			dTask.RefreshTask(&target)
			if target.IsVideoStream {
				return
			}
			if _, ok := task.DownloadSpeedOfProgress.Load(target.FileHash + target.ReqId); !ok {
				pp.Log(ctx, "download has stopped")
				return
			}
			for _, slice := range target.SliceInfo {
				pp.DebugLog(ctx, "taskid ======= ", slice.TaskId)
				if file.CheckSliceExisting(target.FileHash, target.FileName, slice.SliceStorageInfo.SliceHash, target.SavePath, target.ReqId) {
					pp.Log(ctx, "slice exist already,", slice.SliceStorageInfo.SliceHash)
					setDownloadSliceSuccess(ctx, slice.SliceStorageInfo.SliceHash, dTask)
					task.DownloadProgress(ctx, target.FileHash, target.ReqId, slice.SliceStorageInfo.SliceSize)
				} else {
					pp.DebugLog(ctx, "request download data")
					req := requests.ReqDownloadSliceData(&target, slice)
					SendReqDownloadSlice(ctx, target.FileHash, slice, req, target.ReqId)
				}
			}
			pp.DebugLog(ctx, "DownloadFileSlice(&target)", target)
		} else {
			task.DeleteDownloadTask(target.FileHash, target.WalletAddress, task.LOCAL_REQID)
			pp.Log(ctx, "failed to download，", target.Result.Msg)
		}
	}
}
