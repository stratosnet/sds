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

// prepay is a concrete implementation of event
// PP node wants to send a prepay transaction
type prepay struct {
	event
}

const prepayEvent = "prepay"

// GetPrepayHandler creates event and return handler func for it
func GetPrepayHandler(s *net.Server) EventHandleFunc {
	e := prepay{newEvent(prepayEvent, s, prepayCallbackFunc)}
	return e.Handle
}

// prepayCallbackFunc is the main process of broadcasting a prepay transaction
func prepayCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	fmt.Println("Received prepay msg in SP")
	body := message.(*protos.ReqPrepay)

	rsp := &protos.RspPrepay{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
	}

	pp := &table.PP{
		P2pAddress: body.P2PAddress,
	}

	if s.CT.Fetch(pp) != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "Could not find this PP node. Please register first"
		return rsp, header.RspPrepay
	}

	relayMsg := &protos.RelayMessage{
		Type: sds.TypeBroadcast,
		Data: body.Tx,
	}
	msgBytes, err := proto.Marshal(relayMsg)
	if err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "Could not marshal message to send to relay: " + err.Error()
		return rsp, header.RspPrepay
	}

	s.SubscriptionServer.Broadcast("broadcast", msgBytes)
	return rsp, header.RspPrepay
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *prepay) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqPrepay{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
