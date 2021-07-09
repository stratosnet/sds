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

// activated is a concrete implementation of event
// stratoschain createResourceNode transaction success. PP node will become active
type activated struct {
	event
}

const activatedEvent = "activated"

// GetActivatedHandler creates event and return handler func for it
func GetActivatedHandler(s *net.Server) EventHandleFunc {
	e := activated{newEvent(activatedEvent, s, activatedCallbackFunc)}
	return e.Handle
}

// activatedCallbackFunc is the main process of marking the new PP node as active
func activatedCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqActivated)

	rsp := &protos.RspActivated{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
	}

	pp := &table.PP{
		P2PAddress: body.P2PAddress,
	}

	if s.CT.Fetch(pp) != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "Could not find this PP node."
		return rsp, header.RspActivated
	}

	pp.Active = table.PP_ACTIVE
	if err := s.CT.Save(pp); err != nil {
		utils.ErrorLog(err)
	}

	s.SendMsg(body.P2PAddress, header.RspActivated, rsp)
	return rsp, header.RspActivated
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *activated) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqActivated{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
