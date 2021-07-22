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
	"strings"
)

// searchAlbum is a concrete implementation of event
type searchAlbum struct {
	event
}

const searchAlbumEvent = "search_album"

// GetSearchAlbumHandler creates event and return handler func for it
func GetSearchAlbumHandler(s *net.Server) EventHandleFunc {
	e := searchAlbum{newEvent(searchAlbumEvent, s, searchAlbumCallbackFunc)}
	return e.Handle
}

func searchAlbumCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqSearchAlbum)

	rsp := &protos.RspSearchAlbum{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		P2PAddress:    body.P2PAddress,
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
		AlbumInfo:     nil,
	}

	if body.P2PAddress == "" || body.WalletAddress == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "P2P key address and wallet address can't be empty"
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

	if res, err := s.CT.FetchTables([]table.Album{}, params); err == nil {

		albums := res.([]table.Album)
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

	total, _ := s.CT.CountTable(new(table.Album), map[string]interface{}{"where": params["where"]})

	rsp.Total = uint64(total)

	rsp.Page = body.Page

	return rsp, header.RspSearchAlbum
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *searchAlbum) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqSearchAlbum{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
