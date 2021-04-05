package events

import (
	"context"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg/header"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/sp/net"
	"github.com/qsnetwork/sds/sp/storages/data"
	"time"
)

// DownloadTaskInfo
type DownloadTaskInfo struct {
	Server *net.Server
}

// GetServer
func (e *DownloadTaskInfo) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *DownloadTaskInfo) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *DownloadTaskInfo) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqDownloadTaskInfo)

	callback := func(message interface{}) (interface{}, string) {

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
		task := new(data.DownloadTask)
		task.TaskId = taskID
		e.GetServer().Lock()
		if e.GetServer().Load(task) == nil {

			rsp.TaskId = task.TaskId
			rsp.SliceHash = task.SliceHash
			rsp.SliceSize = task.SliceSize
			rsp.StorageWalletAddress = task.StorageWalletAddress
			rsp.WalletAddressList = task.WalletAddressList
			rsp.SliceNumber = task.SliceNumber
			rsp.Time = uint64(time.Now().Unix())

			rsp.Result = &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			}

		} else {
			rsp.Result.Msg = "task doesn't exist"
		}
		e.GetServer().Unlock()

		return rsp, header.RspDownloadTaskInfo
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)

}
