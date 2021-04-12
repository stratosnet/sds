package events

import (
	"context"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
	"time"

	"github.com/google/uuid"
)

// ShareFile
type ShareFile struct {
	Server *net.Server
}

// GetServer
func (e *ShareFile) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *ShareFile) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *ShareFile) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqShareFile)

	callback := func(message interface{}) (interface{}, string) {

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

		userShare := new(table.UserShare)


		if body.FileHash != "" {

			file := new(table.File)
			file.WalletAddress = body.WalletAddress
			file.Hash = body.FileHash
			if e.GetServer().CT.Fetch(file) != nil {
				rsp.Result.State = protos.ResultState_RES_FAIL
				rsp.Result.Msg = "file not exist"
				return rsp, header.RspShareFile
			}
			userShare.Hash = body.FileHash
			userShare.ShareType = table.SHARE_TYPE_FILE
		}


		if body.PathHash != "" {

			directory := new(table.UserDirectory)
			directory.DirHash = body.PathHash
			if e.GetServer().CT.Fetch(directory) != nil {
				rsp.Result.State = protos.ResultState_RES_FAIL
				rsp.Result.Msg = "目录不存在"
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

		if err := e.GetServer().CT.Save(userShare); err != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = err.Error()
			return rsp, header.RspShareFile
		}

		rsp.ShareId = userShare.ShareId
		rsp.SharePassword = userShare.Password
		rsp.ShareLink = userShare.GenerateShareLink(userShare.ShareId, userShare.RandCode)

		return rsp, header.RspShareFile
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
