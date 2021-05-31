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

// activate is a concrete implementation of event
// PP node is trying to become active
type activate struct {
	event
}

const activateEvent = "activate"

// GetActivateHandler creates event and return handler func for it
func GetActivateHandler(s *net.Server) EventHandleFunc {
	e := activate{newEvent(activateEvent, s, activateCallbackFunc)}
	return e.Handle
}

// activateCallbackFunc is the main process of activating a registered PP node
func activateCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	fmt.Println("Received activate msg in SP")
	body := message.(*protos.ReqActivate)

	rsp := &protos.RspActivate{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		AlreadyActive: false,
	}

	pp := &table.PP{
		WalletAddress: body.WalletAddress,
	}

	if s.CT.Fetch(pp) != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "Could not find this PP node. Please register first"
		return rsp, header.RspActivate
	}

	if pp.Active == table.PP_ACTIVE {
		rsp.AlreadyActive = true
		return rsp, header.RspActivate
	}

	relayMsg := &protos.RelayMessage{
		Type: sds.TypeBroadcast,
		Data: body.Tx,
	}
	msgBytes, err := proto.Marshal(relayMsg)
	if err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "Could not marshal message to send to relay: " + err.Error()
		return rsp, header.RspActivate
	}

	s.SubscriptionServer.Broadcast("broadcast", msgBytes)
	return rsp, header.RspActivate
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *activate) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqActivate{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
