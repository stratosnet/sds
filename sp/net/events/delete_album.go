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

// deleteAlbum is a concrete implementation of event
type deleteAlbum struct {
	event
}

const deleteAlbumEvent = "delete_album"

// GetDeleteAlbumHandler creates event and return handler func for it
func GetDeleteAlbumHandler(s *net.Server) EventHandleFunc {
	e := deleteAlbum{newEvent(deleteAlbumEvent, s, deleteAlbumCallbackFunc)}
	return e.Handle
}

// deleteAlbumCallbackFunc is the main process of deleting an album
func deleteAlbumCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqDeleteAlbum)

	rsp := &protos.RspDeleteAlbum{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		P2PAddress:    body.P2PAddress,
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
		AlbumId:       body.AlbumId,
	}

	if body.P2PAddress == "" || body.WalletAddress == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "P2P key address and wallet address can't be empty"
		return rsp, header.RspDeleteAlbum
	}

	if body.AlbumId == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "album ID can't be empty"
		return rsp, header.RspDeleteAlbum
	}

	album := &table.Album{
		AlbumId: body.AlbumId,
	}

	if err := s.CT.Fetch(album); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "album doesn't exist"
		return rsp, header.RspDeleteAlbum
	}

	if album.WalletAddress != body.WalletAddress {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "not authorized to process"
		return rsp, header.RspDeleteAlbum
	}

	if err := s.CT.Trash(album); err != nil {
		return rsp, header.RspDeleteAlbum
	}

	s.CT.GetDriver().Delete("album_has_file", map[string]interface{}{"album_id = ?": body.AlbumId})

	return rsp, header.RspDeleteAlbum
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *deleteAlbum) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqDeleteAlbum{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
