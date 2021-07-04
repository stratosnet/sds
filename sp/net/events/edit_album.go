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
	"unicode/utf8"
)

// editAlbum
type editAlbum struct {
	event
}

const editAlbumEvent = "edit_album"

// GetEditAlbumHandler creates event and return handler func for it
func GetEditAlbumHandler(s *net.Server) EventHandleFunc {
	e := editAlbum{newEvent(editAlbumEvent, s, editAlbumCallbackFunc)}
	return e.Handle
}

// editAlbumCallbackFunc is the main process of edit album
func editAlbumCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqEditAlbum)

	rsp := &protos.RspEditAlbum{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		P2PAddress:    body.P2PAddress,
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
		AlbumId:       body.AlbumId,
	}

	if body.P2PAddress == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "P2P key address can't be empty"
		return rsp, header.RspEditAlbum
	}

	if body.AlbumId == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "album ID can't be empty"
		return rsp, header.RspEditAlbum
	}

	if body.AlbumName == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "album title can't be empty"
		return rsp, header.RspEditAlbum
	}

	if utf8.RuneCountInString(body.AlbumName) > 64 {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "album title is too long"
		return rsp, header.RspEditAlbum
	}

	album := &table.Album{AlbumId: body.AlbumId}

	if s.CT.Fetch(album) != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "album doesn't exist"
		return rsp, header.RspEditAlbum
	}

	s.CT.GetDriver().Delete("album_has_file", map[string]interface{}{"album_id = ?": album.AlbumId})

	for _, f := range body.ChangeFiles {
		if err := album.AddFile(s.CT, f); err != nil {
			utils.ErrorLogf(eventHandleErrorTemplate, editAlbumEvent, "add changed file to db", err)
		}
	}

	var isPrivate byte = table.ALBUM_IS_PUBLIC
	if body.IsPrivate {
		isPrivate = table.ALBUM_IS_PRIVATE
	}

	album.Cover = body.AlbumCoverHash
	album.Name = body.AlbumName
	album.Introduction = body.AlbumBlurb
	album.IsPrivate = isPrivate

	if err := s.CT.Update(album); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "failed to edit album " + err.Error()
		return rsp, header.RspEditAlbum
	}

	return rsp, header.RspEditAlbum
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *editAlbum) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := new(protos.ReqEditAlbum)
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
