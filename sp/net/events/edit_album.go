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

// EditAlbum
type EditAlbum struct {
	Server *net.Server
}

// GetServer
func (e *EditAlbum) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *EditAlbum) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *EditAlbum) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqEditAlbum)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqEditAlbum)

		rsp := &protos.RspEditAlbum{
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
			return rsp, header.RspEditAlbum
		}

		if body.AlbumId == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "album ID can't be empty"
			return rsp, header.RspEditAlbum
		}

		if body.AlbumName == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "album title can't be empty"
			return rsp, header.RspEditAlbum
		}

		if utf8.RuneCountInString(body.AlbumName) > 64 {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "album title is too long"
			return rsp, header.RspEditAlbum
		}

		album := new(table.Album)
		album.AlbumId = body.AlbumId

		if e.GetServer().CT.Fetch(album) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "album doesn't exist"
			return rsp, header.RspEditAlbum
		}

		e.GetServer().CT.GetDriver().Delete("album_has_file", map[string]interface{}{"album_id = ?": album.AlbumId})

		if len(body.ChangeFiles) > 0 {
			for _, file := range body.ChangeFiles {
				album.AddFile(e.GetServer().CT, file)
			}
		}

		var isPrivate byte = table.ALBUM_IS_PUBLIC
		if body.IsPrivate {
			isPrivate = table.ALBUM_IS_PRIVATE
		}

		album.Cover = body.AlbumCoverHash
		album.Name = body.AlbumName
		album.Introduction = body.AlbumBlurb
		album.IsPrivate = isPrivate

		if err := e.GetServer().CT.Update(album); err != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "failed to edit album " + err.Error()
			return rsp, header.RspEditAlbum
		}

		return rsp, header.RspEditAlbum
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
