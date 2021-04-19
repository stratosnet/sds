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

// findMyAlbum is a concrete implementation of event
type findMyAlbum struct {
	event
}

const findMyAlbumEvent = "find_my_album"

// GetFindMyAlbumHandler creates event and return handler func for it
func GetFindMyAlbumHandler(s *net.Server) EventHandleFunc {
	return findMyAlbum{newEvent(findMyAlbumEvent, s, findMyAlbumCallbackFunc)}.Handle
}

// findMyAlbumCallbackFunc is the main process of find my album
func findMyAlbumCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
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

	res, err := s.CT.FetchTables([]table.Album{}, map[string]interface{}{
		"where":  map[string]interface{}{where: args},
		"limit":  int(limit),
		"offset": int((page - 1) * limit),
	})

	albumInfos := make([]*protos.AlbumInfo, 0)
	if err == nil {
		albums := res.([]table.Album)
		for _, album := range albums {

			albumInfo := &protos.AlbumInfo{
				AlbumId:        album.AlbumId,
				AlbumName:      album.Name,
				AlbumBlurb:     album.Introduction,
				AlbumVisit:     int64(album.VisitCount),
				AlbumTime:      album.Time,
				AlbumCoverLink: album.GetCoverLink(s.Conf.FileStorage.PictureLibAddress),
				IsPrivate:      album.IsPrivate == table.ALBUM_IS_PRIVATE,
			}

			albumInfos = append(albumInfos, albumInfo)
		}
	}

	total, _ := s.CT.CountTable(&table.Album{}, map[string]interface{}{"where": map[string]interface{}{where: args}})

	rsp.Total = uint64(total)

	rsp.AlbumInfo = albumInfos

	return rsp, header.RspFindMyAlbum
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *findMyAlbum) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqFindMyAlbum{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
