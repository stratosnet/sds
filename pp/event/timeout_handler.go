package event

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/task"
)

type DownloadTimeoutHandler struct {
}

func (handler *DownloadTimeoutHandler) Handle(ctx context.Context, message *msg.RelayMsgBuf) {
	target := &protos.ReqDownloadSlice{}
	if err := proto.Unmarshal(message.MSGData, target); err != nil {

	}

	dTask, ok := task.GetDownloadTask(target.FileHash, target.WalletAddress, task.LOCAL_REQID)
	if !ok {
		return
	}

	if _, ok := dTask.SuccessSlice[target.SliceInfo.SliceHash]; ok {
		return
	}

	newCtx := core.CreateContextWithParentReqId(ctx, message.MSGHead.ReqId)
	setDownloadSliceFail(newCtx, target.SliceInfo.SliceHash, target.TaskId, target.IsVideoCaching, dTask)
}

func (handler *DownloadTimeoutHandler) GetDuration() time.Duration {
	return 180 * time.Second
}

func (handler *DownloadTimeoutHandler) GetTimeoutMsg(reqMessage *msg.RelayMsgBuf) *msg.RelayMsgBuf {
	return reqMessage
}

func (handler *DownloadTimeoutHandler) CanDelete(rspMessage *msg.RelayMsgBuf) bool {
	var target protos.RspDownloadSlice
	if !requests.UnmarshalMessageData(rspMessage.MSGData, &target) {
		return false
	}
	return target.NeedReport
}
