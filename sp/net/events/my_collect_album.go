package events

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
)

// myCollectAlbum is a concrete implementation of event
type myCollectAlbum struct {
	event
}

const myCollectAlbumEvent = "my_collect_album"

// GetMyCollectAlbumHandler creates event and return handler func for it
func GetMyCollectAlbumHandler(s *net.Server) EventHandleFunc {
	e := myCollectAlbum{newEvent(myCollectAlbumEvent, s, myCollectAlbumCallbackFunc)}
	return e.Handle
}

// myCollectAlbumCallbackFunc is the main process of getting my collect album
func myCollectAlbumCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
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

	res, err := s.CT.FetchTables([]table.Album{}, map[string]interface{}{
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
					AlbumCoverLink: album.GetCoverLink(s.Conf.FileStorage.PictureLibAddress),
					IsPrivate:      album.IsPrivate == table.ALBUM_IS_PRIVATE,
				}
			}
		}
	}

	return rsp, header.RspMyCollectionAlbum
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *myCollectAlbum) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqMyCollectionAlbum{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
