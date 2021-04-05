package events

import (
	"context"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg/header"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/sp/net"
	"github.com/qsnetwork/sds/sp/storages/table"
)

// MyCollectAlbum
type MyCollectAlbum struct {
	Server *net.Server
}

// GetServer
func (e *MyCollectAlbum) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *MyCollectAlbum) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *MyCollectAlbum) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqMyCollectionAlbum)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqMyCollectionAlbum)

		rsp := &protos.RspMyCollectionAlbum{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.WalletAddress,
			ReqId:         body.ReqId,
			Page:          body.Page,
		}

		if body.WalletAddress == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address can't be empty"
			return rsp, header.RspMyCollectionAlbum
		}

		var limit uint64 = 10
		var page uint64 = 1
		if body.Number > 0 {
			limit = body.Number
		}
		if body.Page > 0 {
			page = body.Page
		}

		where := ""
		args := []interface{}{body.WalletAddress}
		if body.AlbumType != protos.AlbumType_ALL {
			where = "a.type = ?"
			args = append(args, int(body.AlbumType))
		}

		if body.Keyword != "" {
			where = where + " AND ( a.name LIKE '%" + body.Keyword + "%' OR a.introduction LIKE '%" + body.Keyword + "%' )"
		}

		res, err := e.GetServer().CT.FetchTables([]table.Album{}, map[string]interface{}{
			"alias":   "a",
			"columns": "a.*",
			"join":    []string{"user_collect_album", "a.album_id = uca.album_id AND uca.wallet_address = ?", "uca"},
			"where":   map[string]interface{}{where: args},
			"limit":   int(limit),
			"offset":  int((page - 1) * limit),
		})

		rsp.AlbumInfo = nil

		if err == nil {
			albums := res.([]table.Album)
			if len(albums) > 0 {
				rsp.AlbumInfo = make([]*protos.AlbumInfo, len(albums))
				for idx, album := range albums {
					rsp.AlbumInfo[idx] = &protos.AlbumInfo{
						AlbumId:        album.AlbumId,
						AlbumTime:      album.Time,
						AlbumVisit:     int64(album.VisitCount),
						AlbumBlurb:     album.Introduction,
						AlbumName:      album.Name,
						AlbumCoverLink: album.GetCoverLink(e.GetServer().Conf.FileStorage.PictureLibAddress),
						IsPrivate:      album.IsPrivate == table.ALBUM_IS_PRIVATE,
					}
				}
			}
		}

		return rsp, header.RspMyCollectionAlbum
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
