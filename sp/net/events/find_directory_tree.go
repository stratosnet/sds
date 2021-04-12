package events

import (
	"context"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
)

// FindDirectoryTree
type FindDirectoryTree struct {
	Server *net.Server
}

// GetServer
func (e *FindDirectoryTree) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *FindDirectoryTree) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *FindDirectoryTree) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqFindDirectoryTree)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqFindDirectoryTree)

		rsp := &protos.RspFindDirectoryTree{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.WalletAddress,
			ReqId:         body.ReqId,
			Directory:     "",
			FileInfo:      nil,
		}

		if body.WalletAddress == "" ||
			body.PathHash == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address or path hash can't be empty"
			return rsp, header.RspFindDirectoryTree
		}

		baseDir := new(table.UserDirectory)
		baseDir.DirHash = body.PathHash // body.Directory
		if e.GetServer().CT.Fetch(baseDir) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "path doesn't exist"
			return rsp, header.RspFindDirectoryTree
		}
		rsp.Directory = baseDir.Path

		rsp.FileInfo = baseDir.RecursFindDirs(e.GetServer().CT)

		files := baseDir.RecursFindFiles(e.GetServer().CT)
		if len(files) > 0 {
			rsp.FileInfo = append(rsp.FileInfo, files...)
		}

		return rsp, header.RspFindDirectoryTree
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
