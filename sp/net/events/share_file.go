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

	"github.com/google/uuid"
)

// shareFile is a concrete implementation of event
type shareFile struct {
	event
}

const shareFileEvent = "share_file"

// GetShareFileHandler creates event and return handler func for it
func GetShareFileHandler(s *net.Server) EventHandleFunc {
	e := shareFile{newEvent(shareFileEvent, s, shareFileCallbackFunc)}
	return e.Handle
}

func shareFileCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqShareFile)

	rsp := &protos.RspShareFile{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
	}

	if body.WalletAddress == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "wallet address can't be empty"
		return rsp, header.RspShareFile
	}

	userShare := &table.UserShare{}

	if body.FileHash != "" {

		file := &table.File{
			Hash:        body.FileHash,
			UserHasFile: table.UserHasFile{WalletAddress: body.WalletAddress},
		}
		if s.CT.Fetch(file) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "file not exist"
			return rsp, header.RspShareFile
		}
		userShare.Hash = body.FileHash
		userShare.ShareType = table.SHARE_TYPE_FILE
	}

	if body.PathHash != "" {

		directory := &table.UserDirectory{DirHash: body.PathHash}

		if err := s.CT.Fetch(directory); err != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "directory does not exist"
			return rsp, header.RspShareFile
		}

		userShare.Hash = directory.DirHash
		userShare.ShareType = table.SHARE_TYPE_DIR
	}

	userShare.ShareId = utils.Get16MD5(uuid.New().String())
	userShare.Time = time.Now().Unix()
	userShare.RandCode = utils.GetRandomString(6)

	if body.IsPrivate {
		userShare.OpenType = table.OPEN_TYPE_PRIVATE
		userShare.Password = utils.GetRandomString(4)
	}
	userShare.WalletAddress = body.WalletAddress

	userShare.Deadline = 0
	if body.ShareTime > 0 {
		userShare.Deadline = body.ShareTime + userShare.Time
	}

	if err := s.CT.Save(userShare); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = err.Error()
		return rsp, header.RspShareFile
	}

	rsp.ShareId = userShare.ShareId
	rsp.SharePassword = userShare.Password
	rsp.ShareLink = userShare.GenerateShareLink(userShare.ShareId, userShare.RandCode)

	return rsp, header.RspShareFile
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *shareFile) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqShareFile{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
