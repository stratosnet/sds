package events

import (
	"context"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"unicode/utf8"
)

// RemoveDirectory
type RemoveDirectory struct {
	Server *net.Server
}

// GetServer
func (e *RemoveDirectory) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *RemoveDirectory) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *RemoveDirectory) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqRemoveDirectory)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqRemoveDirectory)

		rsp := &protos.RspRemoveDirectory{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.WalletAddress,
			ReqId:         body.ReqId,
		}

		if utf8.RuneCountInString(body.Directory) > 512 {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "directory name is too long"
			return rsp, header.RspRemoveDirectory
		}

		if body.WalletAddress == "" ||
			body.Directory == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address or directory can't be empty"
			return rsp, header.RspRemoveDirectory
		}

		directory := &table.UserDirectory{WalletAddress: body.WalletAddress, Path: body.Directory}
		directory.DirHash = directory.GenericHash()
		if e.GetServer().CT.Fetch(directory) == nil {
			if err := e.GetServer().CT.Trash(directory); err != nil {
				rsp.Result.State = protos.ResultState_RES_FAIL
				rsp.Result.Msg = "failed to delete directory：" + err.Error()
				return rsp, header.RspRemoveDirectory
			}

			directory.DeleteFileMap(e.GetServer().CT)
		}

		return rsp, header.RspRemoveDirectory
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
