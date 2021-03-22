package events

import (
	"context"
	"encoding/hex"
	"github.com/qsnetwork/qsds/framework/spbf"
	"github.com/qsnetwork/qsds/msg/header"
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/sp/net"
	"github.com/qsnetwork/qsds/sp/storages/table"
	"github.com/qsnetwork/qsds/utils"
	"github.com/qsnetwork/qsds/utils/crypto"
)

// DeleteFile
type DeleteFile struct {
	Server *net.Server
}

// GetServer
func (e *DeleteFile) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *DeleteFile) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *DeleteFile) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqDeleteFile)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqDeleteFile)

		rsp := &protos.RspDeleteFile{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.WalletAddress,
			ReqId:         body.ReqId,
		}

		if ok, msg := e.Validate(body); !ok {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = msg
			return rsp, header.RspDeleteFile
		}

		file := new(table.File)
		file.Hash = body.FileHash
		file.WalletAddress = body.WalletAddress
		if e.GetServer().CT.Fetch(file) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "file not exist"
			return rsp, header.RspDeleteFile
		}

		e.GetServer().Remove(file.GetCacheKey())

		e.GetServer().CT.DeleteTable(&table.UserHasFile{
			FileHash:      body.FileHash,
			WalletAddress: body.WalletAddress,
		})

		e.GetServer().CT.GetDriver().Delete("user_directory_map_file", map[string]interface{}{
			"wallet_address = ? AND file_hash = ?": []interface{}{
				body.WalletAddress, body.FileHash,
			},
		})

		user := new(table.User)
		user.WalletAddress = body.WalletAddress
		if e.GetServer().CT.Fetch(user) == nil {
		}

		return rsp, header.RspDeleteFile
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}

func (e *DeleteFile) Validate(req *protos.ReqDeleteFile) (bool, string) {

	if req.WalletAddress == "" ||
		req.FileHash == "" {

		return false, "wallet address or filehash can't be empty"
	}

	if len(req.Sign) <= 0 {
		return false, "signature is needed"
	}

	user := &table.User{
		WalletAddress: req.WalletAddress,
	}
	if e.GetServer().CT.Fetch(user) != nil {
		return false, "not authorized to process"
	}

	pukInByte, err := hex.DecodeString(user.Puk)
	if err != nil {
		return false, err.Error()
	}

	puk, err := crypto.UnmarshalPubkey(pukInByte)
	if err != nil {
		return false, err.Error()
	}

	data := req.WalletAddress + req.FileHash
	if !utils.ECCVerify([]byte(data), req.Sign, puk) {
		return false, "signature verification failed"
	}

	return true, ""
}
