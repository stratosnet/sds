package events

import (
	"context"
	"github.com/qsnetwork/qsds/framework/spbf"
	"github.com/qsnetwork/qsds/msg/header"
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/sp/net"
	"github.com/qsnetwork/qsds/sp/storages/table"
	"strings"
)

// SearchAlbum
type SearchAlbum struct {
	Server *net.Server
}

// GetServer
func (e *SearchAlbum) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *SearchAlbum) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *SearchAlbum) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqSearchAlbum)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqSearchAlbum)

		rsp := &protos.RspSearchAlbum{
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
			return rsp, header.RspSearchAlbum
		}

		order := "visit_count DESC"
		if body.AlbumSortType == protos.AlbumSortType_LATEST {
			order = "time DESC"
		}

		where := make([]string, 0)
		args := make([]interface{}, 0)
		if body.AlbumType != protos.AlbumType_ALL {
			//where = append(where, "type = ?", "is_private = ?", "wallet_address != ?")
			where = append(where, "type = ?", "is_private = ?")
			//args = append(args, int(body.AlbumType), table.ALBUM_IS_PUBLIC, body.WalletAddress)
			args = append(args, int(body.AlbumType), table.ALBUM_IS_PUBLIC)
		}

		if body.Keyword != "" {
			where = append(where, "(name LIKE '%"+body.Keyword+"%' OR introduction LIKE '%"+body.Keyword+"%')")
		}

		var limit uint64 = 10
		var page uint64 = 1
		if body.Number > 0 {
			limit = body.Number
		}
		if body.Page > 0 {
			page = body.Page
		}

		params := map[string]interface{}{
			"where":   map[string]interface{}{strings.Join(where, " AND "): args},
			"orderBy": order,
			"limit":   int(limit),
			"offset":  int((page - 1) * limit),
		}

		if res, err := e.GetServer().CT.FetchTables([]table.Album{}, params); err == nil {

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

		total, _ := e.GetServer().CT.CountTable(new(table.Album), map[string]interface{}{"where": params["where"]})

		rsp.Total = uint64(total)

		rsp.Page = body.Page

		return rsp, header.RspSearchAlbum
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
