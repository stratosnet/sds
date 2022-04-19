package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils"
)

func CheckAndSendRetryMessage(dTask *task.DownloadTask) {
	if !dTask.NeedRetry() {
		return
	}
	if f, ok := task.DownloadFileMap.Load(dTask.FileHash); ok {
		fInfo := f.(*protos.RspFileStorageInfo)
		peers.SendMessageToSPServer(requests.ReqDownloadFileWrongData(fInfo, dTask), header.ReqDownloadFileWrong)
	}
}

// RspFileStorageInfo SP-PP , PP-P
func RspDownloadFileWrong(ctx context.Context, conn core.WriteCloser) {
	// PP check whether itself is the storage PP, if not transfer
	utils.Log("get，RspDownloadFileWrong")
	var target protos.RspFileStorageInfo
	if requests.UnmarshalData(ctx, &target) {
		utils.DebugLog("file hash", target.FileHash)
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			utils.Log("download starts: ")
			dTask, ok := task.GetDownloadTask(target.FileHash, target.WalletAddress)
			if !ok {
				utils.DebugLog("cannot find the download task")
				return
			}
			dTask.RefreshTask(&target)
			if target.IsVideoStream {
				return
			}
			if _, ok := task.DownloadSpeedOfProgress.Load(target.FileHash); !ok {
				utils.Log("download has stopped")
				return
			}
			for _, rsp := range target.SliceInfo {
				utils.DebugLog("taskid ======= ", rsp.TaskId)
				if file.CheckSliceExisting(target.FileHash, target.FileName, rsp.SliceStorageInfo.SliceHash, target.SavePath) {
					utils.Log("slice exist already,", rsp.SliceStorageInfo.SliceHash)
					setDownloadSliceSuccess(rsp.SliceStorageInfo.SliceHash, dTask)
					task.DownloadProgress(target.FileHash, rsp.SliceStorageInfo.SliceSize)
				} else {
					utils.DebugLog("request download data")
					req := requests.ReqDownloadSliceData(&target, rsp)
					SendReqDownloadSlice(target.FileHash, rsp, req)
				}
			}
			utils.DebugLog("DownloadFileSlice(&target)", target)
		} else {
			utils.Log("failed to download，", target.Result.Msg)
		}
	}
}
