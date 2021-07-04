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

// abstractAlbum is a concrete implementation of event
type abstractAlbum struct {
	event
}

const abstractAlbumEvent = "abstract album"

// AbstractAlbumHandler creates event and return handler func for it
func AbstractAlbumHandler(server *net.Server) EventHandleFunc {
	e := abstractAlbum{newEvent(abstractAlbumEvent, server, abstractAlbumCallbackFunc)}
	return e.Handle
}

// abstractAlbumCallbackFunc is the main process of abstractAlbum
func abstractAlbumCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {

	body := message.(*protos.ReqAbstractAlbum)

	rsp := &protos.RspAbstractAlbum{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		P2PAddress:      body.P2PAddress,
		WalletAddress:   body.WalletAddress,
		ReqId:           body.ReqId,
		MyAlbum:         nil,
		CollectionAlbum: nil,
	}

	if body.P2PAddress == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "P2P key address can't be empty"
		return rsp, header.RspSearchAlbum
	}

	rsp.MyAlbum = &protos.AlbumNumber{}
	rsp.CollectionAlbum = &protos.AlbumNumber{}

	type abstract struct {
		table.Album
		Type  byte
		Total int64
	}

	res, err := s.CT.FetchTables([]abstract{}, map[string]interface{}{
		"columns": "type, COUNT(*) AS total",
		"where":   map[string]interface{}{"wallet_address = ?": body.WalletAddress},
		"groupBy": "type",
	})

	if err == nil {
		myAlbumAbstract := res.([]abstract)
		for _, ab := range myAlbumAbstract {
			rsp.MyAlbum.All = rsp.MyAlbum.All + ab.Total
			if ab.Type == byte(protos.AlbumType_VIDEO) {
				rsp.MyAlbum.Video = ab.Total
			} else if ab.Type == byte(protos.AlbumType_MUSIC) {
				rsp.MyAlbum.Music = ab.Total
			} else if ab.Type == byte(protos.AlbumType_OTHER) {
				rsp.MyAlbum.Other = ab.Total
			}
		}
	}

	res, err = s.CT.FetchTables([]abstract{}, map[string]interface{}{
		"alias":   "a",
		"columns": "a.type, COUNT(*) AS total",
		"join":    []string{"user_collect_album", "uca.album_id = a.album_id AND uca.wallet_address = ?", "uca"},
		"where":   map[string]interface{}{"": body.WalletAddress},
		"groupBy": "type",
	})

	if err == nil {
		collectAlbumAbstract := res.([]abstract)
		for _, ab := range collectAlbumAbstract {
			rsp.CollectionAlbum.All = rsp.CollectionAlbum.All + ab.Total
			if ab.Type == byte(protos.AlbumType_VIDEO) {
				rsp.CollectionAlbum.Video = ab.Total
			} else if ab.Type == byte(protos.AlbumType_MUSIC) {
				rsp.CollectionAlbum.Music = ab.Total
			} else if ab.Type == byte(protos.AlbumType_OTHER) {
				rsp.CollectionAlbum.Other = ab.Total
			}
		}
	}

	return rsp, header.RspAbstractAlbum
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *abstractAlbum) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		album := &protos.ReqAbstractAlbum{}
		if err := e.handle(ctx, conn, album); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
