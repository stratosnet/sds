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

// albumFileSort is a concrete implementation of event
type albumFileSort struct {
	event
}

const albumFileSortEvent = "album_file_sort"

// GetFileSortHandler creates event and return handler func for it
func GetFileSortHandler(s *net.Server) EventHandleFunc {
	e := albumFileSort{newEvent(albumFileSortEvent, s, getFileSortCallbackFunc)}
	return e.Handle
}

// getFileSortCallbackFunc is the main process of album file sort
func getFileSortCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {

	body := message.(*protos.ReqFileSort)

	rsp := &protos.RspFileSort{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
	}

	if body.WalletAddress == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "wallet address can't be empty"
		return rsp, header.RspFileSort
	}

	if body.AlbumId == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "album ID can't be empty"
		return rsp, header.RspFileSort
	}

	album := &table.Album{AlbumId: body.AlbumId}

	if err := s.CT.Fetch(album); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "album doesn't exist"
		return rsp, header.RspFileSort
	}

	for _, f := range body.Files {
		albumHasFile := &table.AlbumHasFile{
			AlbumId:  body.AlbumId,
			FileHash: f.FileHash,
			Sort:     f.SortId,
			Time:     time.Now().Unix(),
		}
		if _, err := s.CT.UpdateTable(albumHasFile); err != nil {
			utils.ErrorLogf("event handler error: album file sort update table error: %v", err)
		}
	}

	return rsp, header.RspFileSort
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *albumFileSort) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqFileSort{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
