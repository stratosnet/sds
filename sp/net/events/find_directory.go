package events

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
)

// findDirectory is a concrete implementation of event
type findDirectory struct {
	event
}

const findDirectoryEvent = "find_directory"

// GetFindDirectoryHandler creates event and return handler func for it
func GetFindDirectoryHandler(s *net.Server) EventHandleFunc {
	return findDirectory{newEvent(findDirectoryEvent, s, findDirCallbackFunc)}.Handle
}

// findDirCallbackFunc is the main process of finding directory
func findDirCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqFindDirectory)

	rsp := &protos.RspFindDirectory{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
		FileInfo:      nil,
	}

	if body.WalletAddress == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "wallet address can't be empty"
		return rsp, header.RspFindDirectory
	}

	baseDir := &table.UserDirectory{
		WalletAddress: body.WalletAddress,
	}

	rsp.FileInfo = baseDir.RecursFindDirs(s.CT)

	return rsp, header.RspFindDirectory
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *findDirectory) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqFindDirectory{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()

}
