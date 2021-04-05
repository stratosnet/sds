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

// AlbumFileSort
type AlbumFileSort struct {
	Server *net.Server
}

// GetServer
func (e *AlbumFileSort) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *AlbumFileSort) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *AlbumFileSort) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqFileSort)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqFileSort)

		rsp := &protos.RspFileSort{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.WalletAddress,
			ReqId:         body.ReqId,
		}

		if body.WalletAddress == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address can't be empty"
			return rsp, header.RspFileSort
		}

		if body.AlbumId == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "album ID can't be empty"
			return rsp, header.RspFileSort
		}

		album := new(table.Album)
		album.AlbumId = body.AlbumId
		if e.GetServer().CT.Fetch(album) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "album doesn't exist"
			return rsp, header.RspFileSort
		}

		if len(body.Files) > 0 {
			for _, file := range body.Files {
				albumHasFile := new(table.AlbumHasFile)
				albumHasFile.AlbumId = body.AlbumId
				albumHasFile.FileHash = file.FileHash
				albumHasFile.Sort = file.SortId
				albumHasFile.Time = time.Now().Unix()
				e.GetServer().CT.UpdateTable(albumHasFile)
			}
		}

		return rsp, header.RspFileSort
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
