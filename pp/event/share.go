package event

import (
	"context"
	"fmt"
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
					fmt.Println("_______________________________")
					fmt.Println("file_name:", info.Name)
					fmt.Println("file_hash:", info.FileHash)
					fmt.Println("file_size:", info.FileSize)
					fmt.Println("link_time:", info.LinkTime)
					fmt.Println("link_time_exp:", info.LinkTimeExp)
					fmt.Println("ShareId:", info.ShareId)
					fmt.Println("ShareLink:", info.ShareLink)
				}
			} else {
				fmt.Println("action  failed", target.Result.Msg)
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
				fmt.Println("ShareId", target.ShareId)
				fmt.Println("ShareLink", target.ShareLink)
				fmt.Println("SharePassword", target.SharePassword)
			} else {
				fmt.Println("action  failed", target.Result.Msg)
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
				fmt.Println("cancel share success:", target.ShareId)
			} else {
				fmt.Println("action  failed", target.Result.Msg)
			}
			putData(target.ReqId, HTTPDeleteShare, &target)
		} else {
			transferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}

}

// GetShareFile
func GetShareFile(keyword, sharePassword, reqID string, w http.ResponseWriter) {
	utils.DebugLog("GetShareFileGetShareFileGetShareFileGetShareFile")
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
	utils.DebugLog("ReqGetShareFileReqGetShareFileReqGetShareFileReqGetShareFileReqGetShareFileReqGetShareFile")
	transferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspGetShareFile
func RspGetShareFile(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspGetShareFile
	if unmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				fmt.Println("FileInfo:", target.FileInfo)
			} else {
				fmt.Println("action  failed", target.Result.Msg)
			}
			putData(target.ReqId, HTTPGetShareFile, &target)
		} else {
			transferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}

}
