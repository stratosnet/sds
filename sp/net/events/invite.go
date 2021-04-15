package events

import (
	"context"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"time"
)

// Invite
type Invite struct {
	Server *net.Server
}

// GetServer
func (e *Invite) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *Invite) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *Invite) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqInvite)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqInvite)

		rsp := &protos.RspInvite{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.WalletAddress,
			ReqId:         body.ReqId,
		}

		if body.WalletAddress == "" ||
			body.InvitationCode == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address or invitation code can't be empty"
			return rsp, header.RspInvite
		}

		invite := new(table.UserInvite)
		invite.InvitationCode = body.InvitationCode
		if e.GetServer().CT.Fetch(invite) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "invitation code can't be empty"
			return rsp, header.RspInvite
		}

		if invite.Times > 5 {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "invitation code is used up(5 times)"
			return rsp, header.RspInvite
		}

		user := new(table.User)
		user.WalletAddress = body.WalletAddress
		if e.GetServer().CT.Fetch(user) != nil {
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
		rsp.CapacityDelta = e.GetServer().System.InviteReward
		user.Capacity = user.Capacity + rsp.CapacityDelta

		rsp.CurrentCapacity = user.GetCapacity()
		rsp.CapacityDelta = rsp.CapacityDelta / 1048576

		user.BeInvited = 1

		if e.GetServer().CT.Save(user) == nil {

			invite.Times = invite.Times + 1
			if e.GetServer().CT.Update(invite) == nil {

				uir := &table.UserInviteRecord{
					InvitationCode: invite.InvitationCode,
					WalletAddress:  body.WalletAddress,
					Reward:         rsp.CapacityDelta,
					Time:           time.Now().Unix(),
				}
				e.GetServer().CT.StoreTable(uir)
			}
		}

		return rsp, header.RspInvite
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
