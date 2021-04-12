package events

import (
	"context"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
)

// AlbumContent
type AlbumContent struct {
	Server *net.Server
}

// GetServer
func (e *AlbumContent) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *AlbumContent) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *AlbumContent) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqAlbumContent)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqAlbumContent)

		rsp := &protos.RspAlbumContent{
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
			return rsp, header.RspAlbumContent
		}

		if body.AlbumId == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "album ID can't be empty"
			return rsp, header.RspAlbumContent
		}

		album := new(table.Album)
		album.AlbumId = body.AlbumId

		if e.GetServer().CT.Fetch(album) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "album doesn't exist or is deleted"
			return rsp, header.RspAlbumContent
		}

		if body.WalletAddress != album.WalletAddress &&
			album.IsPrivate == table.ALBUM_IS_PRIVATE {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "can't check private album"
			return rsp, header.RspAlbumContent
		}

		rsp.OwnerWalletAddress = album.WalletAddress
		rsp.AlbumInfo = &protos.AlbumInfo{
			AlbumId:        album.AlbumId,
			AlbumName:      album.Name,
			AlbumBlurb:     album.Introduction,
			AlbumTime:      album.Time,
			AlbumVisit:     int64(album.VisitCount),
			AlbumCoverLink: album.GetCoverLink(e.GetServer().Conf.FileStorage.PictureLibAddress),
			AlbumType:      protos.AlbumType(album.Type),
			IsPrivate:      album.IsPrivate == table.ALBUM_IS_PRIVATE,
		}

		userCollectAlbum := new(table.UserCollectAlbum)
		userCollectAlbum.AlbumId = album.AlbumId
		userCollectAlbum.WalletAddress = body.WalletAddress

		rsp.IsCollection = false
		params := map[string]interface{}{
			"where": map[string]interface{}{
				"album_id = ? AND wallet_address = ?": []interface{}{album.AlbumId, body.WalletAddress},
			},
		}
		if e.GetServer().CT.FetchTable(userCollectAlbum, params) == nil {
			rsp.IsCollection = true
		}

		// 刷新访问次数
		album.VisitCount = album.VisitCount + 1
		e.GetServer().CT.Update(album)

		rsp.FileInfo = album.GetFiles(e.GetServer().CT)

		return rsp, header.RspAlbumContent
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
