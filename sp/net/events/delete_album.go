package events

import (
	"context"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
)

// DeleteAlbum
type DeleteAlbum struct {
	Server *net.Server
}

// GetServer
func (e *DeleteAlbum) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *DeleteAlbum) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *DeleteAlbum) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqDeleteAlbum)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqDeleteAlbum)

		rsp := &protos.RspDeleteAlbum{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.WalletAddress,
			ReqId:         body.ReqId,
			AlbumId:       body.AlbumId,
		}

		if body.WalletAddress == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address can't be empty"
			return rsp, header.RspDeleteAlbum
		}

		if body.AlbumId == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "album ID can't be empty"
			return rsp, header.RspDeleteAlbum
		}

		album := new(table.Album)
		album.AlbumId = body.AlbumId
		if e.GetServer().CT.Fetch(album) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "album doesn't exist"
			return rsp, header.RspDeleteAlbum
		}

		if album.WalletAddress != body.WalletAddress {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "not authorized to process"
			return rsp, header.RspDeleteAlbum
		}

		if e.GetServer().CT.Trash(album) == nil {
			e.GetServer().CT.GetDriver().Delete("album_has_file", map[string]interface{}{"album_id = ?": body.AlbumId})
		}

		return rsp, header.RspDeleteAlbum
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
