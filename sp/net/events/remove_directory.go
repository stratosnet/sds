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
	"unicode/utf8"
)

// removeDirectory is a concrete implementation of event
type removeDirectory struct {
	event
}

const rmDirEvent = "remove_directory"

//GetRmDirHandler creates event and return handler func for it
func GetRmDirHandler(s *net.Server) EventHandleFunc {
	e := removeDirectory{newEvent(rmDirEvent, s, rmDirCallbackFunc)}
	return e.Handle
}

// rmDirCallbackFunc is the main process of removing directory
func rmDirCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqRemoveDirectory)

	rsp := &protos.RspRemoveDirectory{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
	}

	if utf8.RuneCountInString(body.Directory) > 512 {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "directory name is too long"
		return rsp, header.RspRemoveDirectory
	}

	if body.WalletAddress == "" || body.Directory == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "wallet address or directory can't be empty"
		return rsp, header.RspRemoveDirectory
	}

	directory := &table.UserDirectory{
		WalletAddress: body.WalletAddress,
		Path:          body.Directory,
	}
	directory.DirHash = directory.GenericHash()

	if err := s.CT.Fetch(directory); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, rmDirEvent, "fetch directory from db", err)
		return rsp, header.RspRemoveDirectory
	}

	if err := s.CT.Trash(directory); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "failed to delete directory：" + err.Error()
		return rsp, header.RspRemoveDirectory
	}

	directory.DeleteFileMap(s.CT)

	return rsp, header.RspRemoveDirectory
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *removeDirectory) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqRemoveDirectory{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()

}
