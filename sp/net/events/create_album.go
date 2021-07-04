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
	"unicode/utf8"

	"github.com/google/uuid"
)

// createAlbum is a concrete implementation of event
type createAlbum struct {
	event
}

const createAlbumEvent = "create_album"

// GetCreateAlbumHandler creates event and return handler func for it
func GetCreateAlbumHandler(s *net.Server) EventHandleFunc {
	e := createAlbum{newEvent(createAlbumEvent, s, createAlbumCallbackFunc)}
	return e.Handle
}

// createAlbumCallbackFunc is the main process of album creation
func createAlbumCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqCreateAlbum)

	rsp := &protos.RspCreateAlbum{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		P2PAddress:    body.P2PAddress,
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
	}

	if body.P2PAddress == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "P2P key address can't be empty"
		return rsp, header.RspCreateAlbum
	}

	if body.AlbumName == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "album title can't be empty"
		return rsp, header.RspCreateAlbum
	}

	if utf8.RuneCountInString(body.AlbumName) > 64 {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "album title is too long"
		return rsp, header.RspCreateAlbum
	}

	albumType := byte(body.AlbumType)
	if body.AlbumType == protos.AlbumType_ALL {
		albumType = byte(body.AlbumType)
	}

	isPrivate := table.ALBUM_IS_PUBLIC
	if body.IsPrivate {
		isPrivate = table.ALBUM_IS_PRIVATE
	}
	album := &table.Album{
		AlbumId:       utils.Get16MD5(uuid.New().String()),
		Name:          body.AlbumName,
		Introduction:  body.AlbumBlurb,
		WalletAddress: body.WalletAddress,
		Cover:         body.AlbumCoverHash,
		VisitCount:    0,
		Time:          time.Now().Unix(),
		Type:          albumType,
		State:         table.STATE_NORMAL,
		IsPrivate:     byte(isPrivate),
	}

	if err := s.CT.Save(album); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = err.Error()
		return rsp, header.RspCreateAlbum
	}

	if len(body.FileInfo) > 0 {
		for _, f := range body.FileInfo {
			if err := album.AddFile(s.CT, f); err != nil {
				utils.ErrorLogf(eventHandleErrorTemplate, createAlbumEvent, "add file", err)
			}
		}
	}

	rsp.AlbumId = album.AlbumId

	return rsp, header.RspCreateAlbum
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *createAlbum) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqCreateAlbum{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
