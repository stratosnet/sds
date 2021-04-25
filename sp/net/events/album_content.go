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

// albumContent is a concrete implementation of event
type albumContent struct {
	event
}

const albumContentEvent = "album_content"

// GetAlbumContentHandler creates event and return handler func for it
func GetAlbumContentHandler(s *net.Server) EventHandleFunc {
	e := albumContent{newEvent(albumContentEvent, s, albumContentCallbackFunc)}
	return e.Handle

}

// albumContentCallbackFunc is the main process of getting album content
func albumContentCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {

	body := message.(*protos.ReqAlbumContent)

	rsp := &protos.RspAlbumContent{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
		AlbumId:       body.AlbumId,
	}

	if body.WalletAddress == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "wallet address can't be empty"
		return rsp, header.RspAlbumContent
	}

	if body.AlbumId == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "album ID can't be empty"
		return rsp, header.RspAlbumContent
	}

	album := &table.Album{AlbumId: body.AlbumId}

	if err := s.CT.Fetch(album); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "album doesn't exist or is deleted"
		return rsp, header.RspAlbumContent
	}

	if body.WalletAddress != album.WalletAddress &&
		album.IsPrivate == table.ALBUM_IS_PRIVATE {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "can't check private album"
		return rsp, header.RspAlbumContent
	}

	rsp.OwnerWalletAddress = album.WalletAddress
	rsp.AlbumInfo = &protos.AlbumInfo{
		AlbumId:        album.AlbumId,
		AlbumName:      album.Name,
		AlbumBlurb:     album.Introduction,
		AlbumTime:      album.Time,
		AlbumVisit:     int64(album.VisitCount),
		AlbumCoverLink: album.GetCoverLink(s.Conf.FileStorage.PictureLibAddress),
		AlbumType:      protos.AlbumType(album.Type),
		IsPrivate:      album.IsPrivate == table.ALBUM_IS_PRIVATE,
	}

	userCollectAlbum := &table.UserCollectAlbum{
		AlbumId:       album.AlbumId,
		WalletAddress: body.WalletAddress,
	}

	rsp.IsCollection = false
	params := map[string]interface{}{
		"where": map[string]interface{}{
			"album_id = ? AND wallet_address = ?": []interface{}{album.AlbumId, body.WalletAddress},
		},
	}
	if s.CT.FetchTable(userCollectAlbum, params) == nil {
		rsp.IsCollection = true
	}

	// update visit count
	album.VisitCount++

	if err := s.CT.Update(album); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, albumContentEvent, "update album in db", err)
	}

	rsp.FileInfo = album.GetFiles(s.CT)

	return rsp, header.RspAlbumContent
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *albumContent) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqAlbumContent{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
