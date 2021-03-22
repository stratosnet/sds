package events

import (
	"context"
	"github.com/qsnetwork/qsds/framework/spbf"
	"github.com/qsnetwork/qsds/msg/header"
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/sp/net"
	"github.com/qsnetwork/qsds/sp/storages/table"
)

// FindDirectory
type FindDirectory struct {
	Server *net.Server
}

// GetServer
func (e *FindDirectory) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *FindDirectory) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *FindDirectory) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqFindDirectory)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqFindDirectory)

		rsp := &protos.RspFindDirectory{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.WalletAddress,
			ReqId:         body.ReqId,
			FileInfo:      nil,
		}

		if body.WalletAddress == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address can't be empty"
			return rsp, header.RspFindDirectory
		}

		baseDir := new(table.UserDirectory)
		baseDir.WalletAddress = body.WalletAddress
		rsp.FileInfo = baseDir.RecursFindDirs(e.GetServer().CT)

		return rsp, header.RspFindDirectory
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
