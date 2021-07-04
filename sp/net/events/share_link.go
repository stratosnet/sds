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

// shareLink is a concrete implementation of event
type shareLink struct {
	event
}

const shareLinkEvent = "share_link"

// GetShareLinkHandler creates event and return handler fun for it
func GetShareLinkHandler(s *net.Server) EventHandleFunc {
	e := shareLink{newEvent(shareLinkEvent, s, shareLinkCallbackFunc)}
	return e.Handle
}

type shareEx struct {
	table.UserShare
	FileSize uint64
	FileName string
	Path     string
}

// shareLinkCallbackFunc is the main process of share link
func shareLinkCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {

	body := message.(*protos.ReqShareLink)

	rsp := &protos.RspShareLink{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		P2PAddress:    body.P2PAddress,
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
		ShareInfo:     make([]*protos.ShareLinkInfo, 0),
	}

	if body.P2PAddress == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "P2P key address can't be empty"
		return rsp, header.RspShareLink
	}

	var shares []shareEx

	res, err := s.CT.FetchTables([]shareEx{}, map[string]interface{}{
		"alias":   "us",
		"columns": "us.*, ud.path",
		"join":    []string{"user_directory", "ud.dir_hash = us.hash", "ud"},
		"where":   map[string]interface{}{"us.wallet_address = ? AND us.share_type = ?": []interface{}{body.WalletAddress, table.SHARE_TYPE_DIR}},
	})

	if err == nil {
		shares = append(shares, res.([]shareEx)...)
	}

	res, err = s.CT.FetchTables([]shareEx{}, map[string]interface{}{
		"alias":   "us",
		"columns": "us.*, f.name AS file_name, f.size AS file_size",
		"join":    []string{"file", "us.hash = f.hash", "f", "left"},
		"where":   map[string]interface{}{"us.wallet_address = ? AND us.share_type = ?": []interface{}{body.WalletAddress, table.SHARE_TYPE_FILE}},
	})

	if err == nil {
		shares = append(shares, res.([]shareEx)...)
	}

	for _, share := range shares {

		shareInfo := &protos.ShareLinkInfo{
			ShareId:            share.ShareId,
			LinkTime:           uint64(share.Time),
			LinkTimeExp:        uint64(share.Deadline),
			FileHash:           share.Hash,
			OwnerWalletAddress: share.WalletAddress,
			ShareLinkPassword:  share.Password,
			IsPrivate:          share.OpenType == table.OPEN_TYPE_PRIVATE,
			ShareLink:          share.RandCode + "_" + share.ShareId,
		}

		if share.ShareType == table.SHARE_TYPE_FILE {
			shareInfo.Name = share.FileName
			shareInfo.FileSize = share.FileSize
			shareInfo.IsDirectory = false
		} else {
			shareInfo.Name = share.Path
			shareInfo.IsDirectory = true
		}
		rsp.ShareInfo = append(rsp.ShareInfo, shareInfo)
	}

	return rsp, header.RspShareLink
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *shareLink) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqShareLink{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
