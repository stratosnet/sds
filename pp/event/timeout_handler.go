package event

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/stratosnet/framework/core"
	"github.com/stratosnet/framework/msg"
	"github.com/stratosnet/framework/utils"
	"github.com/stratosnet/sds-api/protos"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/task"
)

const DOWNLOAD_SLICE_TIMEOUT = 300

type DownloadTimeoutHandler struct {
}

func (handler *DownloadTimeoutHandler) Handle(ctx context.Context, message *msg.RelayMsgBuf) {
	target := &protos.ReqDownloadSlice{}
	if err := proto.Unmarshal(message.MSGBody, target); err != nil {
		utils.ErrorLog(err)
		return
	}

	if target.RspFileStorageInfo == nil {
		return
	}

	dTask, ok := task.GetDownloadTask(target.RspFileStorageInfo.FileHash, target.RspFileStorageInfo.WalletAddress, task.LOCAL_REQID)
	if !ok {
		return
	}

	var slice *protos.DownloadSliceInfo
	for _, slice = range target.RspFileStorageInfo.SliceInfo {
		if slice.SliceNumber == target.SliceNumber {
			break
		}
	}
	if _, ok := dTask.SuccessSlice[slice.SliceStorageInfo.SliceHash]; ok {
		return
	}

	newCtx := core.CreateContextWithParentReqId(ctx, message.MSGHead.ReqId)
	setDownloadSliceFail(newCtx, slice.SliceStorageInfo.SliceHash, target.RspFileStorageInfo.TaskId, task.LOCAL_REQID, dTask)
}

func (handler *DownloadTimeoutHandler) GetDuration() time.Duration {
	return DOWNLOAD_SLICE_TIMEOUT * time.Second
}

func (handler *DownloadTimeoutHandler) GetTimeoutMsg(reqMessage *msg.RelayMsgBuf) *msg.RelayMsgBuf {
	return reqMessage
}

func (handler *DownloadTimeoutHandler) CanDelete(rspMessage *msg.RelayMsgBuf) bool {
	var target protos.RspDownloadSlice
	if !requests.UnmarshalMessageData(rspMessage.MSGBody, &target) {
		return false
	}
	return target.NeedReport
}
