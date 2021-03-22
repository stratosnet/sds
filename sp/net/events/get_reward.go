package events

import (
	"context"
	"github.com/qsnetwork/qsds/framework/spbf"
	"github.com/qsnetwork/qsds/msg/header"
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/sp/net"
	"github.com/qsnetwork/qsds/sp/storages/table"
)

// GetReward
type GetReward struct {
	Server *net.Server
}

// GetServer
func (e *GetReward) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *GetReward) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *GetReward) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqGetReward)

	callback := func(message interface{}) (interface{}, string) {
		body := message.(*protos.ReqGetReward)
		rsp := &protos.RspGetReward{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			ReqId:         body.ReqId,
			WalletAddress: body.WalletAddress,
		}

		if body.WalletAddress == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address can't be empty"
			return rsp, header.RspGetReward
		}

		user := new(table.User)
		user.WalletAddress = body.WalletAddress
		if e.GetServer().CT.Fetch(user) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "need to login first"
			return rsp, header.RspGetReward
		}

		invite := new(table.UserInvite)
		invite.InvitationCode = user.InvitationCode
		if e.GetServer().CT.Fetch(invite) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "invitation code is invalid"
			return rsp, header.RspGetReward
		}

		if invite.Times < 5 {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "invite times not enough(need 5 times)"
			return rsp, header.RspGetReward
		}

		if user.IsUpgrade == 1 {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "already upgrated"
			return rsp, header.RspGetReward
		}

		user.Capacity = user.Capacity + e.GetServer().System.UpgradeReward
		user.IsUpgrade = 1

		if err := e.GetServer().CT.Save(user); err != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = err.Error()
			return rsp, header.RspGetReward
		}

		rsp.CurrentCapacity = user.GetCapacity()

		return rsp, header.RspGetReward
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
