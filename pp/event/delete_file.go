package event

import (
	"context"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

func DeleteFile(ctx context.Context, fileHash string) {
	if setting.CheckLogin() {
		p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, requests.ReqDeleteFileData(fileHash), header.ReqDeleteFile)
	}
}

func ReqDeleteFile(ctx context.Context, conn core.WriteCloser) {
	p2pserver.GetP2pServer(ctx).TransferSendMessageToSPServer(ctx, core.MessageFromContext(ctx))
}

func RspDeleteFile(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspDeleteFile
	if requests.UnmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				pp.Log(ctx, "delete success ", target.Result.Msg)
			} else {
				pp.Log(ctx, "delete failed ", target.Result.Msg)
			}
		} else {
			p2pserver.GetP2pServer(ctx).TransferSendMessageToPPServByP2pAddress(ctx, target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}

// ReqDeleteSlice delete slice sp-pp  or pp-p only works if sent from server to client
func ReqDeleteSlice(ctx context.Context, conn core.WriteCloser) {
	switch conn.(type) {
	case *cf.ClientConn:
		var target protos.ReqDeleteSlice
		if requests.UnmarshalData(ctx, &target) {
			if target.P2PAddress == setting.P2PAddress {
				if file.DeleteSlice(target.SliceHash) != nil {
					requests.RspDeleteSliceData(target.SliceHash, "failed to delete, file not exist", false)
				} else {
					requests.RspDeleteSliceData(target.SliceHash, "delete successfully", true)
				}
			}
		}
	default:
		utils.DebugLog("get a delete msg from client, ERROR!!!!")
	}
}

func RspDeleteSlice(ctx context.Context, conn core.WriteCloser) {
	p2pserver.GetP2pServer(ctx).TransferSendMessageToSPServer(ctx, core.MessageFromContext(ctx))
}
