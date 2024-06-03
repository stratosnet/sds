package event

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/msg"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/sds-msg/protos"
)

const DOWNLOAD_SLICE_TIMEOUT = 60

type DownloadTimeoutHandler struct {
}

func (handler *DownloadTimeoutHandler) TimeoutHandler(ctx context.Context, message *msg.RelayMsgBuf) {
	target := &protos.ReqDownloadSlice{}
	if err := proto.Unmarshal(message.MSGBody, target); err != nil {
		utils.ErrorLog(err)
		return
	}

	if target.RspFileStorageInfo == nil {
		return
	}

	dTask, ok := task.GetDownloadTask(target.RspFileStorageInfo.FileHash + target.RspFileStorageInfo.WalletAddress + task.LOCAL_REQID)
	if !ok {
		return
	}

	newCtx := core.CreateContextWithParentReqId(ctx, message.MSGHead.ReqId)
	CheckAndSendRetryMessage(newCtx, dTask)
}

func (handler *DownloadTimeoutHandler) GetId(msg *msg.RelayMsgBuf, isReq bool) string {
	if isReq {
		target := &protos.ReqDownloadSlice{}
		if err := proto.Unmarshal(msg.MSGBody, target); err != nil {
			utils.ErrorLog(err)
			return ""
		}
		return target.RspFileStorageInfo.FileHash + target.RspFileStorageInfo.WalletAddress + task.LOCAL_REQID
	}

	target := &protos.RspDownloadSlice{}
	if err := proto.Unmarshal(msg.MSGBody, target); err != nil {
		utils.ErrorLog(err)
		return ""
	}
	return target.FileHash + target.WalletAddress + task.LOCAL_REQID
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

func (handler *DownloadTimeoutHandler) Update(key string) bool {
	_, ok := task.GetDownloadTask(key)
	return ok
}

func (handler *DownloadTimeoutHandler) GetType() int {
	return TYPE_RSP_LAST_TOUCH_TIMER
}
