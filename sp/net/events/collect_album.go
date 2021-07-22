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
	"time"
)

// collectAlbum is a concrete implementation of event
type collectAlbum struct {
	event
}

const collectAlbumEvent = "collect_album"

// GetCollectAlbumHandler creates event and return handler func for it
func GetCollectAlbumHandler(s *net.Server) EventHandleFunc {
	e := collectAlbum{newEvent(collectAlbumEvent, s, collectAlbumCallbackFunc)}
	return e.Handle
}

// collectAlbumCallbackFunc is the main process of collecting album
func collectAlbumCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqCollectionAlbum)

	rsp := &protos.RspCollectionAlbum{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		P2PAddress:    body.P2PAddress,
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
		AlbumId:       body.AlbumId,
		IsCollection:  body.IsCollection,
	}

	if body.P2PAddress == "" || body.WalletAddress == "" || body.AlbumId == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "P2P key address, wallet address and album ID can't be empty"
		return rsp, header.RspCollectionAlbum
	}

	album := &table.Album{
		AlbumId: body.AlbumId,
	}

	if err := s.CT.Fetch(album); err != nil {
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

	collect := &table.UserCollectAlbum{}
	err := s.CT.FetchTable(collect, map[string]interface{}{
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
			if _, err = s.CT.StoreTable(userCollectAlbum); err != nil {
				rsp.Result.State = protos.ResultState_RES_FAIL
				rsp.Result.Msg = err.Error()
				return rsp, header.RspCollectionAlbum
			}
		}
	} else {
		if err == nil {
			if _, err = s.CT.DeleteTable(userCollectAlbum); err != nil {
				utils.ErrorLogf(eventHandleErrorTemplate, collectAlbumEvent, "delete user collect album from table", err)
			}
		}
	}

	return rsp, header.RspCollectionAlbum
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *collectAlbum) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqCollectionAlbum{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
