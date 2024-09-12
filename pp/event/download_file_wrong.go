package event

import (
	"context"
	"strings"
	"time"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/msg/header"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/sds-msg/protos"
)

const (
	ResendFailedDownloadDelay     = 5   // Seconds
	ResendFailedDownloadTimeLimit = 120 // Seconds. If this time has elapsed since the start of the download task, don't resend the ReqDownloadFileWrong
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
		pp.DebugLog(ctx, "Download errors occurred, request for help from SP has sent.")
	}
}

func RspDownloadFileWrong(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspDownloadFileWrong")
	var target protos.RspFileStorageInfo
	if err := VerifyMessage(ctx, header.RspDownloadFileWrong, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}

	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	fileReqId, found := getFileReqIdFromContext(ctx)
	if !found {
		pp.DebugLog(ctx, "cannot find the original file request id")
		return
	}

	pp.DebugLog(ctx, "file hash", target.FileHash)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		pp.Log(ctx, "download starts: ")
		dTask, ok := task.GetDownloadTask(target.FileHash + target.WalletAddress + fileReqId)
		if !ok {
			pp.DebugLog(ctx, "cannot find the download task")
			return
		}
		dTask.RefreshTask(&target)
		if _, ok := task.DownloadSpeedOfProgress.Load(target.FileHash + fileReqId); !ok {
			pp.Log(ctx, "download has stopped")
			return
		}
		for _, slice := range target.SliceInfo {
			utils.DebugLog("taskid ======= ", slice.TaskId)
			if file.CheckSliceExisting(target.FileHash, target.FileName, slice.SliceStorageInfo.SliceHash, fileReqId) {
				pp.Log(ctx, "slice exist already,", slice.SliceStorageInfo.SliceHash)
				setDownloadSliceSuccess(ctx, slice.SliceStorageInfo.SliceHash, dTask)
				task.DownloadProgress(ctx, target.FileHash, fileReqId, slice.SliceOffset.SliceOffsetEnd-slice.SliceOffset.SliceOffsetStart)
			} else {
				dTask.AddFailedSlice(slice.SliceStorageInfo.SliceHash)
				task.DownloadSliceProgress.Store(slice.TaskId+slice.SliceStorageInfo.SliceHash+fileReqId, uint64(0))
				req := requests.ReqDownloadSliceData(ctx, &target, slice)
				newCtx := createAndRegisterSliceReqId(ctx, fileReqId)
				SendReqDownloadSlice(newCtx, target.FileHash, slice, req, fileReqId)
			}
		}
	} else {
		dTask, ok := task.GetDownloadTask(target.FileHash + target.WalletAddress + fileReqId)
		if ok && strings.Contains(target.Result.Msg, "cannot find the task") && time.Now().Unix() < dTask.StartTimestamp+ResendFailedDownloadTimeLimit {
			taskMonitorClock.AddJobWithInterval(time.Second*ResendFailedDownloadDelay, func() {
				CheckAndSendRetryMessage(ctx, dTask)
			})
			return
		}

		task.DeleteDownloadTask(target.FileHash, target.WalletAddress, task.LOCAL_REQID)
		task.DownloadResult(ctx, target.FileHash, false, target.Result.Msg)
	}
}
