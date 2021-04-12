package events

import (
	"context"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
)

// GetCapacity
type GetCapacity struct {
	Server *net.Server
}

// GetServer
func (e *GetCapacity) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *GetCapacity) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *GetCapacity) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqGetCapacity)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqGetCapacity)

		rsp := &protos.RspGetCapacity{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.WalletAddress,
			ReqId:         body.ReqId,
			Capacity:      0,
			FreeCapacity:  0,
		}

		if body.WalletAddress == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address can't be empty"
			return rsp, header.RspGetCapacity
		}

		user := &table.User{WalletAddress: body.WalletAddress}
		if e.GetServer().CT.Fetch(user) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "need to login wallet first"
			return rsp, header.RspConfig
		}

		totalUsed, _ := e.GetServer().CT.SumTable(new(table.File), "f.size", map[string]interface{}{
			"alias": "f",
			"join":  []string{"user_has_file", "f.hash = uhf.file_hash", "uhf"},
			"where": map[string]interface{}{"uhf.wallet_address = ?": user.WalletAddress},
		})

		user.UsedCapacity = uint64(totalUsed)
		e.GetServer().CT.Update(user)

		rsp.Capacity = user.GetCapacity()
		rsp.FreeCapacity = user.GetFreeCapacity()

		return rsp, header.RspGetCapacity
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
