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

// invite is a concrete implementation of event
type invite struct {
	event
}

const inviteEvent = "invite"

// GetInviteHandler creates event and return handler func for it
func GetInviteHandler(s *net.Server) EventHandleFunc {
	e := invite{newEvent(inviteEvent, s, inviteCallbackFunc)}
	return e.Handle
}

// inviteCallbackFunc is the main process of inviting
func inviteCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqInvite)

	rsp := &protos.RspInvite{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
	}

	if body.WalletAddress == "" || body.InvitationCode == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "wallet address or invitation code can't be empty"
		return rsp, header.RspInvite
	}

	invite := &table.UserInvite{
		InvitationCode: body.InvitationCode,
	}

	if err := s.CT.Fetch(invite); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "invitation code can't be empty"
		return rsp, header.RspInvite
	}

	if invite.Times > 5 {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "invitation code is used up(5 times)"
		return rsp, header.RspInvite
	}

	user := &table.User{
		WalletAddress: body.WalletAddress,
	}

	if err := s.CT.Fetch(user); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "not registered"
		return rsp, header.RspInvite
	}

	if user.InvitationCode == body.InvitationCode {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "can't invite self"
		return rsp, header.RspInvite
	}

	if user.BeInvited > 0 {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "already invited by others"
		return rsp, header.RspInvite
	}

	// issue reward
	rsp.CapacityDelta = s.System.InviteReward
	user.Capacity = user.Capacity + rsp.CapacityDelta

	rsp.CurrentCapacity = user.GetCapacity()
	rsp.CapacityDelta = rsp.CapacityDelta / 1048576

	user.BeInvited = 1

	if err := s.CT.Save(user); err != nil {
		return rsp, header.RspInvite
	}

	invite.Times = invite.Times + 1
	if err := s.CT.Update(invite); err != nil {
		return rsp, header.RspInvite
	}

	uir := &table.UserInviteRecord{
		InvitationCode: invite.InvitationCode,
		WalletAddress:  body.WalletAddress,
		Reward:         rsp.CapacityDelta,
		Time:           time.Now().Unix(),
	}
	if _, err := s.CT.StoreTable(uir); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, inviteEvent, "store user invite record table to db", err)
	}

	return rsp, header.RspInvite
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *invite) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqInvite{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
