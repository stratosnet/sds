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

// saveFile is a concrete implementation of event
type saveFile struct {
	event
}

const saveFileEvent = "save_file"

// GetSaveFileHandler creates event and return handler func for it
func GetSaveFileHandler(s *net.Server) EventHandleFunc {
	e := saveFile{newEvent(saveFileEvent, s, saveFileCallbackFunc)}
	return e.Handle
}

// saveFileCallbackFunc is the main process of save file
func saveFileCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqSaveFile)

	rsp := &protos.RspSaveFile{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		P2PAddress:    body.P2PAddress,
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
		FilePath:      "",
	}

	if body.P2PAddress == "" || body.WalletAddress == "" || body.FileHash == "" || body.FileOwnerWalletAddress == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "P2P key address, wallet address and file hash can't be empty"
		return rsp, header.RspSaveFile
	}

	file := &table.File{
		Hash: body.FileHash,
		UserHasFile: table.UserHasFile{
			WalletAddress: body.FileOwnerWalletAddress,
		},
	}

	if err := s.CT.Fetch(file); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "file not exist"
		return rsp, header.RspSaveFile
	}

	userHasFile := &table.UserHasFile{}

	err := s.CT.FetchTable(userHasFile, map[string]interface{}{
		"where": map[string]interface{}{
			"wallet_address = ? AND file_hash = ?": []interface{}{body.WalletAddress, body.FileHash},
		},
	})

	if err == nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "file already in the storage"
		return rsp, header.RspSaveFile
	}

	userHasFile.WalletAddress = body.WalletAddress
	userHasFile.FileHash = body.FileHash
	if _, err = s.CT.StoreTable(userHasFile); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = err.Error()
		return rsp, header.RspSaveFile
	}

	return rsp, header.RspSaveFile
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *saveFile) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqSaveFile{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
