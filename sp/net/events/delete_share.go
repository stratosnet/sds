package events

import (
	"context"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg/header"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/sp/net"
	"github.com/qsnetwork/sds/sp/storages/table"
)

// DeleteShare
type DeleteShare struct {
	Server *net.Server
}

// GetServer
func (e *DeleteShare) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *DeleteShare) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *DeleteShare) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqDeleteShare)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqDeleteShare)

		rsp := &protos.RspDeleteShare{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.WalletAddress,
			ReqId:         body.ReqId,
		}

		if body.ShareId == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "share ID can't be empty"
			return rsp, header.RspDeleteShare
		}

		share := new(table.UserShare)
		share.ShareId = body.ShareId

		if e.GetServer().CT.Fetch(share) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "share doesn't exist"
			return rsp, header.RspDeleteShare
		}

		e.GetServer().CT.Trash(share)

		return rsp, header.RspDeleteShare
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
