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
	"time"
)

// makeDirectory is a concrete implementation of event
type makeDirectory struct {
	event
}

const makeDirectoryEvent = "make_directory"

// GetMakeDirHandler creates event and return handler fun for it
func GetMakeDirHandler(s *net.Server) EventHandleFunc {
	e := makeDirectory{newEvent(makeDirectoryEvent, s, makeDirectoryCallbackFunc)}
	return e.Handle
}

// makeDirectoryCallbackFunc is the main process of make directory
func makeDirectoryCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqMakeDirectory)

	rsp := &protos.RspMakeDirectory{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		P2PAddress:    body.P2PAddress,
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
	}

	if body.P2PAddress == "" || body.WalletAddress == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "P2P key address and wallet address can't be empty"
		return rsp, header.RspMakeDirectory
	}

	var err error
	directory := &table.UserDirectory{}

	if directory.Path, err = directory.OptPath(body.Directory); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = err.Error()
		return rsp, header.RspMakeDirectory
	}

	directory.WalletAddress = body.WalletAddress
	directory.DirHash = directory.GenericHash()

	if err = s.CT.Fetch(directory); err == nil {
		return rsp, header.RspMakeDirectory
	}

	directory.Time = time.Now().Unix()

	if err = s.CT.Save(directory); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "failed to save :" + err.Error()
		return rsp, header.RspMakeDirectory
	}

	return rsp, header.RspMakeDirectory
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *makeDirectory) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqMakeDirectory{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
