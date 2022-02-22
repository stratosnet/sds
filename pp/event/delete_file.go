package event

import (
	"context"
	"net/http"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// DeleteFile
func DeleteFile(fileHash, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		peers.SendMessageDirectToSPOrViaPP(requests.ReqDeleteFileData(fileHash, reqID), header.ReqDeleteFile)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqDeleteFile
func ReqDeleteFile(ctx context.Context, conn core.WriteCloser) {
	peers.TransferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspDeleteFile
func RspDeleteFile(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspDeleteFile
	if requests.UnmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				utils.Log("delete success ", target.Result.Msg)
			} else {
				utils.Log("delete failed ", target.Result.Msg)
			}
			putData(target.ReqId, HTTPDeleteFile, &target)
		} else {
			peers.TransferSendMessageToPPServByP2pAddress(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}

// ReqDeleteSlice delete slice sp-pp  or pp-p only works if sent from server to client
func ReqDeleteSlice(ctx context.Context, conn core.WriteCloser) {
	switch conn.(type) {
	case *cf.ClientConn:
		{
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
		}

	default:
		utils.DebugLog("get a delete msg from client, ERROR!!!!")
		break
	}

}

// RspDeleteSlice RspDeleteSlice
func RspDeleteSlice(ctx context.Context, conn core.WriteCloser) {
	peers.TransferSendMessageToSPServer(core.MessageFromContext(ctx))
}
