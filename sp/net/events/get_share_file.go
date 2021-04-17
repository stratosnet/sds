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
)

// getShareFile is a concrete implementation of event
type getShareFile struct {
	event
}

const getShareFileEvent = "get_share_file"

// GetGetShareFileHandler creates event and return handler func for it
func GetGetShareFileHandler(s *net.Server) EventHandleFunc {
	return getShareFile{newEvent(getShareFileEvent, s, getShareFileCallbackFunc)}.Handle
}

func getShareFileCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
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

		f, err := getFileFromServer(s, body.Keyword)
		if err != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = err.Error()
			return rsp, header.RspGetShareFile
		}
		fileInfos = []*protos.FileInfo{f}

	} else if len(body.Keyword) == 42 {

		files, err := getWalletAddressFiles(s, body.Keyword)
		if err != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = err.Error()
			return rsp, header.RspGetShareFile
		}
		fileInfos = files

	} else {

		share := &table.UserShare{}

		randCode, shareId := share.ParseShareLink(body.Keyword)

		if randCode == "" || shareId == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "failed to parse share link"
			return rsp, header.RspGetShareFile
		}

		share.RandCode = randCode
		share.ShareId = shareId
		if err := s.CT.Fetch(share); err != nil {
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

		f, err := share.GetShareContent(s.CT)
		if err != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = err.Error()
			return rsp, header.RspGetShareFile
		}

		fileInfos = []*protos.FileInfo{f}
	}

	rsp.FileInfo = fileInfos

	return rsp, header.RspGetShareFile

}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *getShareFile) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqGetShareFile{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}

type file struct {
	table.File
	Path          string
	IsPrivate     byte
	WalletAddress string
	ShareId       string
	RandCode      string
	Time          int64
}

// getFileFromServer
func getFileFromServer(s *net.Server, fileHash string) (*protos.FileInfo, error) {

	f := &file{}
	err := s.CT.FetchTable(f, map[string]interface{}{
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
		FileHash:           f.Hash,
		FileName:           f.Name,
		FileSize:           f.Size,
		IsPrivate:          f.IsPrivate == table.OPEN_TYPE_PRIVATE,
		IsDirectory:        false,
		StoragePath:        f.Path,
		OwnerWalletAddress: f.WalletAddress,
		ShareLink:          (&table.UserShare{}).GenerateShareLink(f.ShareId, f.RandCode),
		CreateTime:         uint64(f.Time),
	}, nil

}

func getWalletAddressFiles(s *net.Server, walletAddress string) ([]*protos.FileInfo, error) {
	user := new(table.User)
	user.WalletAddress = walletAddress
	fileInfos := user.GetShareDirs(s.CT)
	files := user.GetShareFiles(s.CT)
	if len(files) > 0 {
		fileInfos = append(fileInfos, files...)
	}
	return fileInfos, nil
}
