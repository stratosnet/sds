package events

import (
	"context"
	"github.com/qsnetwork/qsds/framework/spbf"
	"github.com/qsnetwork/qsds/msg/header"
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/sp/net"
	"github.com/qsnetwork/qsds/sp/storages/table"
)

// ShareLink
type ShareLink struct {
	Server *net.Server
}

// GetServer
func (e *ShareLink) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *ShareLink) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *ShareLink) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqShareLink)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqShareLink)

		rsp := &protos.RspShareLink{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.WalletAddress,
			ReqId:         body.ReqId,
			ShareInfo:     make([]*protos.ShareLinkInfo, 0),
		}

		if body.WalletAddress == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address can't be empty"
			return rsp, header.RspShareLink
		}

		type ShareEx struct {
			table.UserShare
			FileSize uint64
			FileName string
			Path     string
		}

		shares := make([]ShareEx, 0)

		res, err := e.GetServer().CT.FetchTables([]ShareEx{}, map[string]interface{}{
			"alias":   "us",
			"columns": "us.*, ud.path",
			"join":    []string{"user_directory", "ud.dir_hash = us.hash", "ud"},
			"where":   map[string]interface{}{"us.wallet_address = ? AND us.share_type = ?": []interface{}{body.WalletAddress, table.SHARE_TYPE_DIR}},
		})

		if err == nil {
			shareDirs := res.([]ShareEx)
			if len(shareDirs) > 0 {
				shares = append(shares, shareDirs...)
			}
		}

		res, err = e.GetServer().CT.FetchTables([]ShareEx{}, map[string]interface{}{
			"alias":   "us",
			"columns": "us.*, f.name AS file_name, f.size AS file_size",
			"join":    []string{"file", "us.hash = f.hash", "f", "left"},
			"where":   map[string]interface{}{"us.wallet_address = ? AND us.share_type = ?": []interface{}{body.WalletAddress, table.SHARE_TYPE_FILE}},
		})

		if err == nil {
			shareFiles := res.([]ShareEx)
			if len(shareFiles) > 0 {
				shares = append(shares, shareFiles...)
			}
		}

		if len(shares) > 0 {
			for _, share := range shares {

				shareInfo := new(protos.ShareLinkInfo)

				shareInfo.IsPrivate = false
				if share.OpenType == table.OPEN_TYPE_PRIVATE {
					shareInfo.IsPrivate = true
				}
				shareInfo.ShareId = share.ShareId
				shareInfo.LinkTime = uint64(share.Time)
				shareInfo.LinkTimeExp = uint64(share.Deadline)
				shareInfo.FileHash = share.Hash
				shareInfo.OwnerWalletAddress = share.WalletAddress
				shareInfo.ShareLinkPassword = share.Password
				if share.ShareType == table.SHARE_TYPE_FILE {
					shareInfo.Name = share.FileName
					shareInfo.FileSize = share.FileSize
					shareInfo.IsDirectory = false
				} else {
					shareInfo.Name = share.Path
					shareInfo.IsDirectory = true
				}
				shareInfo.ShareLink = share.RandCode + "_" + share.ShareId
				rsp.ShareInfo = append(rsp.ShareInfo, shareInfo)
			}
		}

		return rsp, header.RspShareLink
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
