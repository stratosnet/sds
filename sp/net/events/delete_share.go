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

// deleteShare is a concrete implementation of event
type deleteShare struct {
	event
}

const deleteShareEvent = "delete_share"

// GetDeleteShareHandler creates event and return handler func for it
func GetDeleteShareHandler(s *net.Server) EventHandleFunc {
	e := deleteShare{newEvent(deleteShareEvent, s, deleteShareCallbackFunc)}
	return e.Handle
}

// deleteShareCallbackFunc is the main process of delete share
func deleteShareCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqDeleteShare)

	rsp := &protos.RspDeleteShare{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		P2PAddress:    body.P2PAddress,
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
	}

	if body.ShareId == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "share ID can't be empty"
		return rsp, header.RspDeleteShare
	}

	share := &table.UserShare{ShareId: body.ShareId}

	if err := s.CT.Fetch(share); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "share doesn't exist"
		return rsp, header.RspDeleteShare
	}

	if err := s.CT.Trash(share); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, deleteShareEvent, "trash share from db", err)
	}

	return rsp, header.RspDeleteShare
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *deleteShare) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqDeleteShare{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
