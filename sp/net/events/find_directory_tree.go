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

// findDirectoryTree is a concrete implementation of event
type findDirectoryTree struct {
	event
}

const findDirTreeEvent = "find_directory_tree"

// GetFindDirectoryTreeHandler creates event and return handler func for it
func GetFindDirectoryTreeHandler(s *net.Server) EventHandleFunc {
	e := findDirectoryTree{newEvent(findDirTreeEvent, s, findDirTreeCallbackFunc)}
	return e.Handle
}

// findDirTreeCallbackFunc is the main process of finding directory tree
func findDirTreeCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqFindDirectoryTree)

	rsp := &protos.RspFindDirectoryTree{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		P2PAddress:    body.P2PAddress,
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
		Directory:     "",
		FileInfo:      nil,
	}

	if body.P2PAddress == "" || body.WalletAddress == "" || body.PathHash == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "P2P key address, wallet address and path hash can't be empty"
		return rsp, header.RspFindDirectoryTree
	}

	baseDir := &table.UserDirectory{
		DirHash: body.PathHash, // body.Directory
	}

	if err := s.CT.Fetch(baseDir); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "path doesn't exist"
		return rsp, header.RspFindDirectoryTree
	}

	rsp.Directory = baseDir.Path
	rsp.FileInfo = baseDir.RecursFindDirs(s.CT)

	files := baseDir.RecursFindFiles(s.CT)
	rsp.FileInfo = append(rsp.FileInfo, files...)

	return rsp, header.RspFindDirectoryTree
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *findDirectoryTree) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqFindDirectoryTree{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
