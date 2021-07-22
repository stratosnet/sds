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

// deactivated is a concrete implementation of event
// stratoschain removeResourceNode transaction success. PP node will become inactive
type deactivated struct {
	event
}

const deactivatedEvent = "deactivated"

// GetDeactivateHandler creates event and return handler func for it
func GetDeactivatedHandler(s *net.Server) EventHandleFunc {
	e := deactivated{newEvent(deactivatedEvent, s, deactivatedCallbackFunc)}
	return e.Handle
}

// deactivatedCallbackFunc is the main process of marking the PP node as inactive
func deactivatedCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqDeactivated)

	rsp := &protos.RspDeactivated{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
	}

	pp := &table.PP{
		P2pAddress: body.P2PAddress,
	}

	if s.CT.Fetch(pp) != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "Could not find this PP node."
		return rsp, header.RspDeactivated
	}

	pp.Active = table.PP_INACTIVE
	if err := s.CT.Save(pp); err != nil {
		utils.ErrorLog(err)
	}

	s.SendMsg(body.P2PAddress, header.RspDeactivated, rsp)
	return rsp, header.RspDeactivated
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *deactivated) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqDeactivated{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
