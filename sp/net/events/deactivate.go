package events

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/relay/sds"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
)

// deactivate is a concrete implementation of event
// An active PP node wants to become inactive
type deactivate struct {
	event
}

const deactivateEvent = "deactivate"

// GetDeactivateHandler creates event and return handler func for it
func GetDeactivateHandler(s *net.Server) EventHandleFunc {
	e := deactivate{newEvent(deactivateEvent, s, deactivateCallbackFunc)}
	return e.Handle
}

// deactivateCallbackFunc is the main process of deactivating an active PP node
func deactivateCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	fmt.Println("Received deactivate msg in SP")
	body := message.(*protos.ReqDeactivate)

	rsp := &protos.RspDeactivate{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		ActivationState: table.PP_ACTIVE,
	}

	pp := &table.PP{
		WalletAddress: body.WalletAddress,
	}

	if s.CT.Fetch(pp) != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "Could not find this PP node. Please register first"
		return rsp, header.RspDeactivate
	}

	if pp.Active == table.PP_INACTIVE {
		rsp.ActivationState = table.PP_INACTIVE
		return rsp, header.RspDeactivate
	}

	relayMsg := &protos.RelayMessage{
		Type: sds.TypeBroadcast,
		Data: body.Tx,
	}
	msgBytes, err := proto.Marshal(relayMsg)
	if err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "Could not marshal message to send to relay: " + err.Error()
		return rsp, header.RspDeactivate
	}

	s.SubscriptionServer.Broadcast("broadcast", msgBytes)
	return rsp, header.RspDeactivate
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *deactivate) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqDeactivate{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
