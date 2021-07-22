package events

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/data"
	"github.com/stratosnet/sds/utils"
	"time"
)

// downloadTaskInfo is a concrete implementation of event
type downloadTaskInfo struct {
	event
}

const downloadTaskInfoEvent = "download_task_info"

// GetDownloadTaskInfoHandler creates event and return handler func for it
func GetDownloadTaskInfoHandler(s *net.Server) EventHandleFunc {
	e := downloadTaskInfo{newEvent(downloadTaskInfoEvent, s, downloadTaskInfoCallbackFunc)}
	return e.Handle
}

// downloadTaskInfoCallbackFunc is the main process of download task info
func downloadTaskInfoCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqDownloadTaskInfo)

	rsp := &protos.RspDownloadTaskInfo{
		Result: &protos.Result{
			State: protos.ResultState_RES_FAIL,
		},
		Id: body.Id,
	}

	taskID := body.TaskId

	if taskID == "" {
		rsp.Result.Msg = "task ID can't be empty"
	}

	// todo change to read from redis
	task := &data.DownloadTask{
		TaskId: taskID,
	}

	s.Lock()
	defer s.Unlock()
	if err := s.Load(task); err != nil {
		rsp.Result.Msg = "task doesn't exist"
		return rsp, header.RspDownloadTaskInfo
	}

	rsp.TaskId = task.TaskId
	rsp.SliceHash = task.SliceHash
	rsp.SliceSize = task.SliceSize
	rsp.StorageP2PAddress = task.StorageP2PAddress
	rsp.P2PAddressList = task.P2PAddressList
	rsp.SliceNumber = task.SliceNumber
	rsp.Time = uint64(time.Now().Unix())

	rsp.Result = &protos.Result{
		State: protos.ResultState_RES_SUCCESS,
	}

	return rsp, header.RspDownloadTaskInfo
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *downloadTaskInfo) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqDownloadTaskInfo{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()

}
