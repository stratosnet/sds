package events

import (
	"context"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
)

// SaveFile
type SaveFile struct {
	Server *net.Server
}

// GetServer
func (e *SaveFile) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *SaveFile) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *SaveFile) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqSaveFile)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqSaveFile)

		rsp := &protos.RspSaveFile{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.WalletAddress,
			ReqId:         body.ReqId,
			FilePath:      "",
		}

		if body.WalletAddress == "" ||
			body.FileHash == "" ||
			body.FileOwnerWalletAddress == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address or filehash can't be empty"
			return rsp, header.RspSaveFile
		}

		file := new(table.File)
		file.Hash = body.FileHash
		file.WalletAddress = body.FileOwnerWalletAddress
		if e.GetServer().CT.Fetch(file) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "file not exist"
			return rsp, header.RspSaveFile
		}

		userHasFile := new(table.UserHasFile)

		err := e.GetServer().CT.FetchTable(userHasFile, map[string]interface{}{
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
		if ok, err := e.GetServer().CT.StoreTable(userHasFile); !ok {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = err.Error()
			return rsp, header.RspSaveFile
		}

		return rsp, header.RspSaveFile
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
