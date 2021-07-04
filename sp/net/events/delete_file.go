package events

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
)

// deleteFile is a concrete implementation of event
type deleteFile struct {
	event
}

const deleteFileEvent = "deleteFileEvent"

// GetDeleteFileHandler creates event and return handler func for it
func GetDeleteFileHandler(s *net.Server) EventHandleFunc {
	e := deleteFile{newEvent(deleteFileEvent, s, deleteFileCallbackFunc)}
	return e.Handle
}

// deleteFileCallbackFunc is the main process of delete file
func deleteFileCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqDeleteFile)

	rsp := &protos.RspDeleteFile{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		P2PAddress:    body.P2PAddress,
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
	}

	if ok, msg := validateDeleteFileRequest(s, body); !ok {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = msg
		return rsp, header.RspDeleteFile
	}

	file := &table.File{
		Hash: body.FileHash,
		UserHasFile: table.UserHasFile{
			WalletAddress: body.WalletAddress,
		},
	}

	if s.CT.Fetch(file) != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "file not exist"
		return rsp, header.RspDeleteFile
	}

	if err := s.Remove(file.GetCacheKey()); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, deleteFileEvent, "remove file from db", err)
	}

	if _, err := s.CT.DeleteTable(&table.UserHasFile{FileHash: body.FileHash, WalletAddress: body.WalletAddress}); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, deleteFileEvent, "delete user file table from db", err)
	}

	_ = s.CT.GetDriver().Delete("user_directory_map_file", map[string]interface{}{
		"wallet_address = ? AND file_hash = ?": []interface{}{
			body.WalletAddress, body.FileHash,
		},
	})

	user := &table.User{P2PAddress: body.P2PAddress}

	if err := s.CT.Fetch(user); err != nil {
		return rsp, header.RspDeleteFile
	}
	// todo ?

	return rsp, header.RspDeleteFile
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *deleteFile) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqDeleteFile{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}

// validateDeleteFileRequest validate request
func validateDeleteFileRequest(s *net.Server, req *protos.ReqDeleteFile) (bool, string) {

	if req.P2PAddress == "" || req.FileHash == "" {

		return false, "P2P key address and file hash can't be empty"
	}

	if len(req.Sign) <= 0 {
		return false, "signature is needed"
	}

	user := &table.User{
		P2PAddress: req.P2PAddress,
	}
	if s.CT.Fetch(user) != nil {
		return false, "not authorized to process"
	}

	puk, err := hex.DecodeString(user.Puk)
	if err != nil {
		return false, err.Error()
	}

	data := req.P2PAddress + req.FileHash
	if !ed25519.Verify(puk, []byte(data), req.Sign) {
		return false, "signature verification failed"
	}

	return true, ""
}
