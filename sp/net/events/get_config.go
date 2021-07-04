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

// getConfig is a concrete implementation of event
type getConfig struct {
	event
}

const getConfigEvent = "get_config"

// GetGetConfigHandler creates event and return handler func for it
func GetGetConfigHandler(s *net.Server) EventHandleFunc {
	e := getConfig{newEvent(getConfigEvent, s, getConfigCallbackFunc)}
	return e.Handle
}

// getConfigCallbackFunc is the main process of get configuration
func getConfigCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqConfig)
	rsp := &protos.RspConfig{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
			Msg:   "request success",
		},
		ReqId:         body.ReqId,
		P2PAddress:    body.P2PAddress,
		WalletAddress: body.WalletAddress,
	}

	user := &table.User{P2PAddress: body.P2PAddress}
	if s.CT.Fetch(user) != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "need to login wallet first"
		return rsp, header.RspConfig
	}

	rsp.InvitationCode = user.InvitationCode

	userInvite := &table.UserInvite{InvitationCode: user.InvitationCode}

	if s.CT.Fetch(userInvite) == nil {
		rsp.Invite = uint64(userInvite.Times)
	}

	rsp.IsUpgrade = user.IsUpgrade != 0
	rsp.Capacity = user.GetCapacity()
	rsp.FreeCapacity = user.GetFreeCapacity()

	return rsp, header.RspConfig
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *getConfig) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqConfig{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
