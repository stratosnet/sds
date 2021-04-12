package events

import (
	"context"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
)

// GetConfig
type GetConfig struct {
	Server *net.Server
}

// GetServer
func (e *GetConfig) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *GetConfig) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *GetConfig) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqConfig)

	callback := func(message interface{}) (interface{}, string) {
		body := message.(*protos.ReqConfig)
		rsp := &protos.RspConfig{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
				Msg:   "request success",
			},
			ReqId:         body.ReqId,
			WalletAddress: body.WalletAddress,
		}

		user := &table.User{WalletAddress: body.WalletAddress}
		if e.GetServer().CT.Fetch(user) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "need to login wallet first"
			return rsp, header.RspConfig
		}

		rsp.InvitationCode = user.InvitationCode

		userInvite := new(table.UserInvite)
		userInvite.InvitationCode = user.InvitationCode
		if e.GetServer().CT.Fetch(userInvite) == nil {
			rsp.Invite = uint64(userInvite.Times)
		}

		if user.IsUpgrade == 0 {
			rsp.IsUpgrade = false
		} else {
			rsp.IsUpgrade = true
		}

		rsp.Capacity = user.GetCapacity()
		rsp.FreeCapacity = user.GetFreeCapacity()

		return rsp, header.RspConfig
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
