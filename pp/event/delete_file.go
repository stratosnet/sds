package event

import (
	"context"
	"fmt"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"net/http"
)

// DeleteFile
func DeleteFile(fileHash, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		sendMessage(client.PPConn, reqDeleteFileData(fileHash, reqID), header.ReqDeleteFile)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqDeleteFile
func ReqDeleteFile(ctx context.Context, conn core.WriteCloser) {
	transferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspDeleteFile
func RspDeleteFile(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspDeleteFile
	if unmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				fmt.Println("delete success ", target.Result.Msg)
			} else {
				fmt.Println("delete failed ", target.Result.Msg)
			}
			putData(target.ReqId, HTTPDeleteFile, &target)
		} else {
			transferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}

// ReqDeleteSlice delete slice sp-pp  or pp-p only works if sent from server to client
func ReqDeleteSlice(ctx context.Context, conn core.WriteCloser) {
	switch conn.(type) {
	case *cf.ClientConn:
		{
			var target protos.ReqDeleteSlice
			if unmarshalData(ctx, &target) {
				if target.P2PAddress == setting.P2PAddress {
					if file.DeleteSlice(target.SliceHash) != nil {
						rspDeleteSliceData(target.SliceHash, "failed to delete, file not exist", false)
					} else {
						rspDeleteSliceData(target.SliceHash, "delete successfully", true)
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
	transferSendMessageToSPServer(core.MessageFromContext(ctx))
}
