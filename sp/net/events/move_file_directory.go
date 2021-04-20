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

// moveFileDirectory is a concrete implementation of event
type moveFileDirectory struct {
	event
}

const mvFileDirEvent = "move_file_directory"

// GetMoveFileDirHandler creates event and return handler func for it
func GetMoveFileDirHandler(s *net.Server) EventHandleFunc {
	e := moveFileDirectory{newEvent(mvFileDirEvent, s, moveFileDirCallbackFunc)}
	return e.Handle
}

// moveFileDirCallbackFunc is the main process of move file directory
func moveFileDirCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqMoveFileDirectory)

	rsp := &protos.RspMoveFileDirectory{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
		FilePath:      "",
	}

	if body.WalletAddress == "" || body.FileHash == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "wallet address or  file hash can't be empty"
		return rsp, header.RspMoveFileDirectory
	}

	if body.DirectoryOriginal == body.DirectoryTarget {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "target directory can't be original"
		return rsp, header.RspMoveFileDirectory
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
		return rsp, header.RspMoveFileDirectory
	}

	originDir := &table.UserDirectory{}

	if body.DirectoryOriginal != "" {
		originDir.WalletAddress = body.WalletAddress
		pathOk, err := originDir.OptPath(body.DirectoryOriginal)
		if err != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wrong original file path: " + err.Error()
			return rsp, header.RspMoveFileDirectory
		}

		originDir.Path = pathOk
		originDir.DirHash = originDir.GenericHash()
		if err = s.CT.Fetch(originDir); err != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wrong original file path"
			return rsp, header.RspMoveFileDirectory
		}
	}

	desDir := &table.UserDirectory{}
	if body.DirectoryTarget != "" {
		desDir.WalletAddress = body.WalletAddress
		pathOk, err := desDir.OptPath(body.DirectoryTarget)
		if err != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wrong target file path: " + err.Error()
			return rsp, header.RspMoveFileDirectory
		}
		desDir.Path = pathOk
		desDir.DirHash = desDir.GenericHash()
		if err := s.CT.Fetch(desDir); err != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wrong target file path"
			return rsp, header.RspMoveFileDirectory
		}
	}

	// if target is empty, then move to root directory, delete user_directory_map_file
	if body.DirectoryTarget == "" || body.DirectoryOriginal != "" {
		needRemoveFileMap := &table.UserDirectoryMapFile{
			DirHash:  originDir.DirHash,
			FileHash: file.Hash,
		}
		if _, err := s.CT.DeleteTable(needRemoveFileMap); err != nil {
			utils.ErrorLogf(eventHandleErrorTemplate, mvFileDirEvent, "delete move file table from db", err)
		}
	}

	// if original directory is empty, then move from root directory to target
	if body.DirectoryOriginal == "" || body.DirectoryTarget != "" {
		fileMap := &table.UserDirectoryMapFile{
			DirHash:  desDir.DirHash,
			FileHash: file.Hash,
			Owner:    body.WalletAddress,
		}
		if _, err := s.CT.StoreTable(fileMap); err != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = err.Error()
			return rsp, header.RspMoveFileDirectory
		}
	}

	return rsp, header.RspMoveFileDirectory
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *moveFileDirectory) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqMoveFileDirectory{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
