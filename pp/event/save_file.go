package event

import (
	"context"
	"fmt"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/setting"
	"net/http"
)

// SaveOthersFile SaveOthersFile
func SaveOthersFile(fileHash, ownerAddress, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		sendMessage(client.PPConn, reqSaveFileData(fileHash, reqID, ownerAddress), header.ReqSaveFile)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqSaveFile ReqSaveFile
func ReqSaveFile(ctx context.Context, conn core.WriteCloser) {
	transferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspSaveFile RspSaveFile
func RspSaveFile(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspSaveFile
	if unmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				fmt.Println("action  successfully", target.Result.Msg)
			} else {
				fmt.Println("action  failed", target.Result.Msg)
			}
			putData(target.ReqId, HTTPMVdir, &target)
		} else {
			transferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}

// SaveFolder SaveFolder
func SaveFolder(folderHash, ownerAddress, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		sendMessage(client.PPConn, reqSaveFolderData(folderHash, reqID, ownerAddress), header.ReqSaveFolder)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqSaveFolder ReqSaveFolder
func ReqSaveFolder(ctx context.Context, conn core.WriteCloser) {
	transferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspSaveFolder RspSaveFolder
func RspSaveFolder(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspSaveFolder
	if unmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				fmt.Println("action  successfully", target.Result.Msg)
			} else {
				fmt.Println("action  failed", target.Result.Msg)
			}
			putData(target.ReqId, HTTPSaveFolder, &target)
		} else {
			transferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}
