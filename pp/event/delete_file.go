package event

import (
	"context"
	"fmt"
	"github.com/qsnetwork/qsds/framework/client/cf"
	"github.com/qsnetwork/qsds/framework/spbf"
	"github.com/qsnetwork/qsds/msg/header"
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/pp/client"
	"github.com/qsnetwork/qsds/pp/file"
	"github.com/qsnetwork/qsds/pp/setting"
	"github.com/qsnetwork/qsds/utils"
	"net/http"
)

// DeleteFile
func DeleteFile(fileHash, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		sendMessage(client.PPConn, reqDeleteFileData(fileHash, reqID), header.ReqDeleteFile)
		stroeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqDeleteFile
func ReqDeleteFile(ctx context.Context, conn spbf.WriteCloser) {
	transferSendMessageToSPServer(spbf.MessageFromContext(ctx))
}

// RspDeleteFile
func RspDeleteFile(ctx context.Context, conn spbf.WriteCloser) {
	var target protos.RspDeleteFile
	if unmarshalData(ctx, &target) {
		if target.WalletAddress == setting.WalletAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				fmt.Println("删除成功 ", target.Result.Msg)

			} else {

				fmt.Println("删除失败 ", target.Result.Msg)

			}
			utils.DebugLog("aaaaaa>>>>>>", target.ReqId)
			putData(target.ReqId, HTTPDeleteFile, &target)
		} else {
			transferSendMessageToClient(target.WalletAddress, spbf.MessageFromContext(ctx))
		}
	}
}

// ReqDeleteSlice delete slice sp-pp  or pp-p only works if sent from server to client
func ReqDeleteSlice(ctx context.Context, conn spbf.WriteCloser) {
	switch conn.(type) {
	case *cf.ClientConn:
		{
			var target protos.ReqDeleteSlice
			if unmarshalData(ctx, &target) {
				if target.WalletAddress == setting.WalletAddress {
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
func RspDeleteSlice(ctx context.Context, conn spbf.WriteCloser) {
	transferSendMessageToSPServer(spbf.MessageFromContext(ctx))
}
