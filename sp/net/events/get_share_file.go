package events

import (
	"context"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"time"
)

// GetShareFile
type GetShareFile struct {
	Server *net.Server
}

// GetServer
func (e *GetShareFile) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *GetShareFile) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *GetShareFile) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqGetShareFile)

	callback := func(message interface{}) (interface{}, string) {
		body := message.(*protos.ReqGetShareFile)
		rsp := &protos.RspGetShareFile{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
				Msg:   "request success",
			},
			ReqId:         body.ReqId,
			WalletAddress: body.WalletAddress,
			FileInfo:      nil,
			IsPrivate:     false,
		}

		if body.Keyword == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "keyword can't be empty"
			return rsp, header.RspGetShareFile
		}

		if body.WalletAddress == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address can't be empty"
			return rsp, header.RspGetShareFile
		}

		var fileInfos []*protos.FileInfo

		if len(body.Keyword) == 64 {

			file, err := e.GetFile(body.Keyword)
			if err != nil {
				rsp.Result.State = protos.ResultState_RES_FAIL
				rsp.Result.Msg = err.Error()
				return rsp, header.RspGetShareFile
			}
			fileInfos = []*protos.FileInfo{file}

		} else if len(body.Keyword) == 42 {

			files, err := e.GetWalletAddressFiles(body.Keyword)
			if err != nil {
				rsp.Result.State = protos.ResultState_RES_FAIL
				rsp.Result.Msg = err.Error()
				return rsp, header.RspGetShareFile
			}
			fileInfos = files

		} else {

			share := new(table.UserShare)

			randCode, shareId := share.ParseShareLink(body.Keyword)

			if randCode == "" || shareId == "" {
				rsp.Result.State = protos.ResultState_RES_FAIL
				rsp.Result.Msg = "failed to parse share link"
				return rsp, header.RspGetShareFile
			}

			share.RandCode = randCode
			share.ShareId = shareId
			if e.GetServer().CT.Fetch(share) != nil {
				rsp.Result.State = protos.ResultState_RES_FAIL
				rsp.Result.Msg = "share doesn't exist"
				return rsp, header.RspGetShareFile
			}

			if share.Deadline > 0 && share.Deadline < time.Now().Unix() {
				rsp.Result.State = protos.ResultState_RES_FAIL
				rsp.Result.Msg = "share is expired"
				return rsp, header.RspGetShareFile
			}

			if share.OpenType == table.OPEN_TYPE_PRIVATE {
				if body.SharePassword == "" {
					rsp.IsPrivate = true
					rsp.Result.State = protos.ResultState_RES_FAIL
					rsp.Result.Msg = "open private file, share password can't be empty"
					return rsp, header.RspGetShareFile
				}
				if body.SharePassword != share.Password {
					rsp.Result.State = protos.ResultState_RES_FAIL
					rsp.Result.Msg = "wrong share password"
					return rsp, header.RspGetShareFile
				}
			}

			file, err := share.GetShareContent(e.GetServer().CT)
			if err != nil {
				rsp.Result.State = protos.ResultState_RES_FAIL
				rsp.Result.Msg = err.Error()
				return rsp, header.RspGetShareFile
			}

			fileInfos = []*protos.FileInfo{file}
		}

		rsp.FileInfo = fileInfos

		return rsp, header.RspGetShareFile
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}

// GetFile
func (e *GetShareFile) GetFile(fileHash string) (*protos.FileInfo, error) {

	type ShareFile struct {
		table.File
		Path          string
		IsPrivate     byte
		WalletAddress string
		ShareId       string
		RandCode      string
		Time          int64
	}

	shareFile := new(ShareFile)
	err := e.GetServer().CT.FetchTable(shareFile, map[string]interface{}{
		"alias":   "f",
		"columns": "f.*, ud.path, us.time, us.is_private, us.wallet_address, us.rand_code, us.share_id",
		"join": [][]string{
			{"user_directory_map_file", "f.hash = udmf.file_hash AND f.hash = ?", "udmf"},
			{"user_directory", "udmf.dir_hash = ud.dir_hash", "ud"},
			{"user_share", "us.hash = f.hash AND us.share_type = ? AND us.open_type = ?", "us"},
		},
		"where": map[string]interface{}{"": []interface{}{fileHash, table.SHARE_TYPE_FILE, table.OPEN_TYPE_PUBLIC}},
	})

	if err != nil {
		return nil, nil
	}

	return &protos.FileInfo{
		FileHash:           shareFile.Hash,
		FileName:           shareFile.Name,
		FileSize:           shareFile.Size,
		IsPrivate:          shareFile.IsPrivate == table.OPEN_TYPE_PRIVATE,
		IsDirectory:        false,
		StoragePath:        shareFile.Path,
		OwnerWalletAddress: shareFile.WalletAddress,
		ShareLink:          new(table.UserShare).GenerateShareLink(shareFile.ShareId, shareFile.RandCode),
		CreateTime:         uint64(shareFile.Time),
	}, nil

}

// GetFile
func (e *GetShareFile) GetWalletAddressFiles(walletAddress string) ([]*protos.FileInfo, error) {
	user := new(table.User)
	user.WalletAddress = walletAddress
	fileInfos := user.GetShareDirs(e.GetServer().CT)
	files := user.GetShareFiles(e.GetServer().CT)
	if len(files) > 0 {
		fileInfos = append(fileInfos, files...)
	}
	return fileInfos, nil
}
