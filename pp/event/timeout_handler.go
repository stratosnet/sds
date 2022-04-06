package event

import (
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/task"
)

type DownloadTimeoutHandler struct {
}

func (handler *DownloadTimeoutHandler) Handle(message *msg.RelayMsgBuf) {
	target := &protos.ReqDownloadSlice{}
	if err := proto.Unmarshal(message.MSGData, target); err != nil {

	}

	dTask, ok := task.GetDownloadTask(target.FileHash, target.WalletAddress)
	if !ok {
		return
	}

	if _, ok := dTask.SuccessSlice[target.SliceInfo.SliceHash]; ok {
		return
	}

	setDownloadSliceFail(target.SliceInfo.SliceHash, dTask)

	if target.IsVideoCaching {
		videoCacheKeep(target.FileHash, target.TaskId)
	}
}

func (handler *DownloadTimeoutHandler) GetDuration() time.Duration {
	return 120 * time.Second
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
