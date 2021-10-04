package event

import (
	"context"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"net/http"
)

// GetAllShareLink GetShareLink
func GetAllShareLink(reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		sendMessage(client.PPConn, reqShareLinkData(reqID), header.ReqShareLink)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// GetReqShareFile GetReqShareFile
func GetReqShareFile(reqID, fileHash, pathHash string, shareTime int64, isPrivate bool, w http.ResponseWriter) {
	if setting.CheckLogin() {
		sendMessage(client.PPConn, reqShareFileData(reqID, fileHash, pathHash, isPrivate, shareTime), header.ReqShareFile)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// DeleteShare DeleteShare
func DeleteShare(shareID, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		sendMessage(client.PPConn, reqDeleteShareData(reqID, shareID), header.ReqDeleteShare)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqShareLink
func ReqShareLink(ctx context.Context, conn core.WriteCloser) {
	// pp send to SP
	utils.DebugLog("ReqShareLinkReqShareLinkReqShareLinkReqShareLink")
	transferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspShareLink
func RspShareLink(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("RspShareLink(ctx context.Context, conn core.WriteCloser) {RspShareLink(ctx context.Context, conn core.WriteCloser) {")
	var target protos.RspShareLink
	if unmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				for _, info := range target.ShareInfo {
					utils.Log("_______________________________")
					utils.Log("file_name:", info.Name)
					utils.Log("file_hash:", info.FileHash)
					utils.Log("file_size:", info.FileSize)
					utils.Log("link_time:", info.LinkTime)
					utils.Log("link_time_exp:", info.LinkTimeExp)
					utils.Log("ShareId:", info.ShareId)
					utils.Log("ShareLink:", info.ShareLink)
				}
			} else {
				utils.Log("action failed", target.Result.Msg)
			}
			putData(target.ReqId, HTTPShareLink, &target)
		} else {
			transferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}

// ReqShareFile
func ReqShareFile(ctx context.Context, conn core.WriteCloser) {
	// pp send to SP
	transferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspShareFile
func RspShareFile(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspShareFile
	if unmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				utils.Log("ShareId", target.ShareId)
				utils.Log("ShareLink", target.ShareLink)
				utils.Log("SharePassword", target.SharePassword)
			} else {
				utils.Log("action failed", target.Result.Msg)
			}
			putData(target.ReqId, HTTPShareFile, &target)
		} else {
			transferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}

// ReqDeleteShare
func ReqDeleteShare(ctx context.Context, conn core.WriteCloser) {
	// pp send to SP
	transferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspDeleteShare
func RspDeleteShare(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspDeleteShare
	if unmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				utils.Log("cancel share success:", target.ShareId)
			} else {
				utils.Log("action failed", target.Result.Msg)
			}
			putData(target.ReqId, HTTPDeleteShare, &target)
		} else {
			transferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}

}

// GetShareFile
func GetShareFile(keyword, sharePassword, reqID string, w http.ResponseWriter) {
	utils.DebugLog("GetShareFile for file ", keyword)
	if setting.CheckLogin() {
		sendMessage(client.PPConn, reqGetShareFileData(keyword, sharePassword, reqID), header.ReqGetShareFile)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqGetShareFile
func ReqGetShareFile(ctx context.Context, conn core.WriteCloser) {
	// pp send to SP
	utils.DebugLog("ReqGetShareFile: transferring message to SP server")
	transferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspGetShareFile
func RspGetShareFile(ctx context.Context, _ core.WriteCloser) {
	var target protos.RspGetShareFile
	if !unmarshalData(ctx, &target) {
		return
	}

	if target.ShareRequest.P2PAddress != setting.P2PAddress {
		transferSendMessageToClient(target.ShareRequest.P2PAddress, core.MessageFromContext(ctx))
		return
	}

	utils.Log("get RspGetShareFile", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		return
	}

	utils.Log("FileInfo:", target.FileInfo)
	putData(target.ShareRequest.ReqId, HTTPGetShareFile, &target)

	for _, fileInfo := range target.FileInfo {
		filePath := "spb://" + fileInfo.OwnerWalletAddress + "/" + fileInfo.FileHash
		sendMessage(client.PPConn, reqFileStorageInfoData(filePath, "", "", false, target.ShareRequest), header.ReqFileStorageInfo)
	}
}
