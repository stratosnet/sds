package events

import (
	"context"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"time"
)

// CollectAlbum
type CollectAlbum struct {
	Server *net.Server
}

// GetServer
func (e *CollectAlbum) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *CollectAlbum) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *CollectAlbum) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqCollectionAlbum)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqCollectionAlbum)

		rsp := &protos.RspCollectionAlbum{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.WalletAddress,
			ReqId:         body.ReqId,
			AlbumId:       body.AlbumId,
			IsCollection:  body.IsCollection,
		}

		if body.WalletAddress == "" ||
			body.AlbumId == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address or album ID can't be empty"
			return rsp, header.RspCollectionAlbum
		}

		album := new(table.Album)
		album.AlbumId = body.AlbumId
		if e.GetServer().CT.Fetch(album) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "album doesn't exist"
			return rsp, header.RspCollectionAlbum
		}

		if album.WalletAddress == body.WalletAddress {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "can't collect own album"
			return rsp, header.RspCollectionAlbum
		}

		if album.IsPrivate == table.ALBUM_IS_PRIVATE {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "can't collect private album"
			return rsp, header.RspCollectionAlbum
		}

		collect := new(table.UserCollectAlbum)
		err := e.GetServer().CT.FetchTable(collect, map[string]interface{}{
			"where": map[string]interface{}{
				"wallet_address = ? AND album_id = ?": []interface{}{
					body.WalletAddress, body.AlbumId,
				},
			},
		})

		userCollectAlbum := &table.UserCollectAlbum{
			WalletAddress: body.WalletAddress,
			AlbumId:       body.AlbumId,
			Time:          time.Now().Unix(),
		}

		if body.IsCollection {
			if err != nil {
				if ok, err := e.GetServer().CT.StoreTable(userCollectAlbum); !ok {
					rsp.Result.State = protos.ResultState_RES_FAIL
					rsp.Result.Msg = err.Error()
					return rsp, header.RspCollectionAlbum
				}
			}
		} else {
			if err == nil {
				e.GetServer().CT.DeleteTable(userCollectAlbum)
			}
		}

		return rsp, header.RspCollectionAlbum
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
