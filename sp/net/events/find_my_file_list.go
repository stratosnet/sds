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

// findMyFileList is a concrete implementation of event
type findMyFileList struct {
	event
}

const findMyFileListEvent = "find_my_file_list"

// GetFindMyFileListHandler creates event and return handler func for it
func GetFindMyFileListHandler(s *net.Server) EventHandleFunc {
	e := findMyFileList{newEvent(findMyFileListEvent, s, findMyFileListCallbackFunc)}
	return e.Handle
}

// findMyFileListCallbackFunc is the main process this find my file list event
func findMyFileListCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqFindMyFileList)

	rsp := &protos.RspFindMyFileList{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		FileInfo:      nil,
		P2PAddress:    body.P2PAddress,
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
	}

	if body.P2PAddress == "" || body.WalletAddress == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "P2P key address and wallet address can't be empty"
		return rsp, header.RspFindMyFileList
	}

	fileInfos := make([]*protos.FileInfo, 0)

	// case body.Directory == "" AND body.FileName == "": query all file under root directory
	// case body.Directory != "" AND body.FileName == "": query all file under specified directory
	// case body.Directory == "" AND body.FileName != "": query specified file under root directory
	// case body.Directory != "" AND body.FileName != "": query specified file under specified directory

	dir := &table.UserDirectory{}

	// body.FileName is empty, query directory
	if body.FileName == "" {
		dirs := dir.FindDirs(s.CT, body.WalletAddress, body.Directory)
		if len(dirs) > 0 {
			fileInfos = append(fileInfos, dirs...)
		}
	}

	// query file
	files := dir.FindFiles(s.CT, body.WalletAddress, body.Directory, body.FileName, "", body.Keyword, body.FileType, body.IsUp)
	if len(files) > 0 {
		fileInfos = append(fileInfos, files...)
	}

	rsp.FileInfo = fileInfos

	return rsp, header.RspFindMyFileList
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *findMyFileList) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqFindMyFileList{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
