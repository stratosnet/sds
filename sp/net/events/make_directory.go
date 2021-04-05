package events

import (
	"context"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg/header"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/sp/net"
	"github.com/qsnetwork/sds/sp/storages/table"
	"time"
)

// MakeDirectory
type MakeDirectory struct {
	Server *net.Server
}

// GetServer
func (e *MakeDirectory) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *MakeDirectory) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *MakeDirectory) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqMakeDirectory)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqMakeDirectory)

		rsp := &protos.RspMakeDirectory{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.WalletAddress,
			ReqId:         body.ReqId,
		}

		if body.WalletAddress == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address can't be empty"
			return rsp, header.RspMakeDirectory
		}

		var err error
		directory := new(table.UserDirectory)
		if directory.Path, err = directory.OptPath(body.Directory); err != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = err.Error()
			return rsp, header.RspMakeDirectory
		}
		directory.WalletAddress = body.WalletAddress

		directory.DirHash = directory.GenericHash()

		if e.GetServer().CT.Fetch(directory) != nil {
			directory.Time = time.Now().Unix()
			if err := e.GetServer().CT.Save(directory); err != nil {
				rsp.Result.State = protos.ResultState_RES_FAIL
				rsp.Result.Msg = "failed to save :" + err.Error()
				return rsp, header.RspMakeDirectory
			}
		}

		return rsp, header.RspMakeDirectory
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
