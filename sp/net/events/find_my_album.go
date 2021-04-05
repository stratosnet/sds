package events

import (
	"context"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg/header"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/sp/net"
	"github.com/qsnetwork/sds/sp/storages/table"
)

// FindMyAlbum
type FindMyAlbum struct {
	Server *net.Server
}

// GetServer
func (e *FindMyAlbum) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *FindMyAlbum) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *FindMyAlbum) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqFindMyAlbum)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqFindMyAlbum)

		rsp := &protos.RspFindMyAlbum{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.WalletAddress,
			ReqId:         body.ReqId,
			AlbumInfo:     nil,
		}

		if body.WalletAddress == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address can't be empty"
			return rsp, header.RspFindMyAlbum
		}

		where := "wallet_address = ?"
		args := []interface{}{body.WalletAddress}
		if body.AlbumType != protos.AlbumType_ALL {
			where = where + " AND type = ?"
			args = append(args, int(body.AlbumType))
		}

		if body.Keyword != "" {
			where = where + " AND ( name LIKE '%" + body.Keyword + "%' OR introduction LIKE '%" + body.Keyword + "%' )"
		}

		var limit uint64 = 10
		var page uint64 = 1
		if body.Number > 0 {
			limit = body.Number
		}
		if body.Page > 0 {
			page = body.Page
		}

		res, err := e.GetServer().CT.FetchTables([]table.Album{}, map[string]interface{}{
			"where":  map[string]interface{}{where: args},
			"limit":  int(limit),
			"offset": int((page - 1) * limit),
		})

		albumInfos := make([]*protos.AlbumInfo, 0)
		if err == nil {
			albums := res.([]table.Album)
			if len(albums) > 0 {
				for _, album := range albums {
					albumInfos = append(albumInfos, &protos.AlbumInfo{
						AlbumId:        album.AlbumId,
						AlbumName:      album.Name,
						AlbumBlurb:     album.Introduction,
						AlbumVisit:     int64(album.VisitCount),
						AlbumTime:      album.Time,
						AlbumCoverLink: album.GetCoverLink(e.GetServer().Conf.FileStorage.PictureLibAddress),
						IsPrivate:      album.IsPrivate == table.ALBUM_IS_PRIVATE,
					})
				}
			}
		}

		total, _ := e.GetServer().CT.CountTable(new(table.Album), map[string]interface{}{"where": map[string]interface{}{where: args}})

		rsp.Total = uint64(total)

		rsp.AlbumInfo = albumInfos

		return rsp, header.RspFindMyAlbum
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
