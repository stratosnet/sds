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

// prepaid is a concrete implementation of event
// stratoschain prepay transaction success
type prepaid struct {
	event
}

const prepaidEvent = "prepaid"

// GetPrepaidHandler creates event and return handler func for it
func GetPrepaidHandler(s *net.Server) EventHandleFunc {
	e := prepaid{newEvent(prepaidEvent, s, prepaidCallbackFunc)}
	return e.Handle
}

// prepaidCallbackFunc is the main process of updating the user capacity following a successful prepay transaction
func prepaidCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqPrepaid)

	rsp := &protos.RspPrepaid{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
	}

	pp := &table.PP{
		WalletAddress: body.WalletAddress,
	}

	if s.CT.Fetch(pp) != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "Could not find this PP node."
		return rsp, header.RspActivated
	}

	// TODO: update capacity
	if err := s.CT.Save(pp); err != nil {
		utils.ErrorLog(err)
	}

	s.SendMsg(body.WalletAddress, header.RspPrepaid, rsp)
	return rsp, header.RspPrepaid
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *prepaid) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqPrepaid{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
