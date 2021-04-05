package events

import (
	"context"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg/header"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/sp/net"
	"github.com/qsnetwork/sds/sp/storages/table"
	"github.com/qsnetwork/sds/utils"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// CreateAlbum
type CreateAlbum struct {
	Server *net.Server
}

// GetServer
func (e *CreateAlbum) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *CreateAlbum) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *CreateAlbum) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqCreateAlbum)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqCreateAlbum)

		rsp := &protos.RspCreateAlbum{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.WalletAddress,
			ReqId:         body.ReqId,
		}

		if body.WalletAddress == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address can't be empty"
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

		var isPrivate byte = table.ALBUM_IS_PUBLIC
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
			IsPrivate:     isPrivate,
		}

		if err := e.GetServer().CT.Save(album); err != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = err.Error()
			return rsp, header.RspCreateAlbum
		}

		if len(body.FileInfo) > 0 {
			for _, file := range body.FileInfo {
				album.AddFile(e.GetServer().CT, file)
			}
		}

		rsp.AlbumId = album.AlbumId

		return rsp, header.RspCreateAlbum
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
