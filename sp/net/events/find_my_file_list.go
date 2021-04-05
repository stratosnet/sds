package events

import (
	"context"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg/header"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/sp/net"
	"github.com/qsnetwork/sds/sp/storages/table"
)

// FindMyFileList
type FindMyFileList struct {
	Server *net.Server
}

// GetServer
func (e *FindMyFileList) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *FindMyFileList) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *FindMyFileList) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqFindMyFileList)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqFindMyFileList)

		rsp := &protos.RspFindMyFileList{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			FileInfo:      nil,
			WalletAddress: body.WalletAddress,
			ReqId:         body.ReqId,
		}

		if body.WalletAddress == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address can't be empty"
			return rsp, header.RspFindMyFileList
		}

		fileInfos := make([]*protos.FileInfo, 0)

		// case body.Directory == "" AND body.FileName == "": query all file under root directory
		// case body.Directory != "" AND body.FileName == "": query all file under specified directory
		// case body.Directory == "" AND body.FileName != "": query specified file under root directory
		// case body.Directory != "" AND body.FileName != "": query specified file under specified directory

		dir := new(table.UserDirectory)

		// body.FileName is empty, query directory
		if body.FileName == "" {
			dirs := dir.FindDirs(e.GetServer().CT, body.WalletAddress, body.Directory)
			if len(dirs) > 0 {
				fileInfos = append(fileInfos, dirs...)
			}
		}

		// query file
		files := dir.FindFiles(e.GetServer().CT, body.WalletAddress, body.Directory, body.FileName, "", body.Keyword, body.FileType, body.IsUp)
		if len(files) > 0 {
			fileInfos = append(fileInfos, files...)
		}

		rsp.FileInfo = fileInfos

		return rsp, header.RspFindMyFileList
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
