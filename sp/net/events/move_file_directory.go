package events

import (
	"context"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg/header"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/sp/net"
	"github.com/qsnetwork/sds/sp/storages/table"
)

// MoveFileDirectory
type MoveFileDirectory struct {
	Server *net.Server
}

// GetServer
func (e *MoveFileDirectory) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *MoveFileDirectory) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *MoveFileDirectory) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqMoveFileDirectory)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqMoveFileDirectory)

		rsp := &protos.RspMoveFileDirectory{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.WalletAddress,
			ReqId:         body.ReqId,
			FilePath:      "",
		}

		if body.WalletAddress == "" ||
			body.FileHash == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address or  filehash can't be empty"
			return rsp, header.RspMoveFileDirectory
		}

		if body.DirectoryOriginal == body.DirectoryTarget {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "target directory can't be original"
			return rsp, header.RspMoveFileDirectory
		}

		file := new(table.File)
		file.Hash = body.FileHash
		file.WalletAddress = body.WalletAddress
		if e.GetServer().CT.Fetch(file) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "file not exist"
			return rsp, header.RspMoveFileDirectory
		}

		originDir := new(table.UserDirectory)
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
			if e.GetServer().CT.Fetch(originDir) != nil {
				rsp.Result.State = protos.ResultState_RES_FAIL
				rsp.Result.Msg = "wrong original file path"
				return rsp, header.RspMoveFileDirectory
			}
		}

		desDir := new(table.UserDirectory)
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
			if e.GetServer().CT.Fetch(desDir) != nil {
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
			e.GetServer().CT.DeleteTable(needRemoveFileMap)
		}

		// if original directory is empty, then move from root directory to target
		if body.DirectoryOriginal == "" || body.DirectoryTarget != "" {
			fileMap := &table.UserDirectoryMapFile{
				DirHash:  desDir.DirHash,
				FileHash: file.Hash,
				Owner:    body.WalletAddress,
			}
			if ok, err := e.GetServer().CT.StoreTable(fileMap); !ok {
				rsp.Result.State = protos.ResultState_RES_FAIL
				rsp.Result.Msg = err.Error()
				return rsp, header.RspMoveFileDirectory
			}
		}

		return rsp, header.RspMoveFileDirectory
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
