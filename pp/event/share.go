package event

import (
	"context"
	"net/http"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/datamesh"
)

// GetAllShareLink GetShareLink
func GetAllShareLink(reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		peers.SendMessageDirectToIndexNodeOrViaPP(requests.ReqShareLinkData(reqID), header.ReqShareLink)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// GetReqShareFile GetReqShareFile
func GetReqShareFile(reqID, fileHash, pathHash string, shareTime int64, isPrivate bool, w http.ResponseWriter) {
	if setting.CheckLogin() {
		peers.SendMessageDirectToIndexNodeOrViaPP(requests.ReqShareFileData(reqID, fileHash, pathHash, isPrivate, shareTime), header.ReqShareFile)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// DeleteShare DeleteShare
func DeleteShare(shareID, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		peers.SendMessageDirectToIndexNodeOrViaPP(requests.ReqDeleteShareData(reqID, shareID), header.ReqDeleteShare)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqShareLink
func ReqShareLink(ctx context.Context, conn core.WriteCloser) {
	// pp send to SP
	utils.DebugLog("ReqShareLinkReqShareLinkReqShareLinkReqShareLink")
	peers.TransferSendMessageToIndexNodeServer(core.MessageFromContext(ctx))
}

// RspShareLink
func RspShareLink(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("RspShareLink(ctx context.Context, conn core.WriteCloser) {RspShareLink(ctx context.Context, conn core.WriteCloser) {")
	var target protos.RspShareLink
	if requests.UnmarshalData(ctx, &target) {
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
				utils.ErrorLog("all share failed:", target.Result.Msg)
			}
			putData(target.ReqId, HTTPShareLink, &target)
		} else {
			peers.TransferSendMessageToPPServByP2pAddress(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}

// ReqShareFile
func ReqShareFile(ctx context.Context, conn core.WriteCloser) {
	// pp send to SP
	peers.TransferSendMessageToIndexNodeServer(core.MessageFromContext(ctx))
}

// RspShareFile
func RspShareFile(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspShareFile
	if requests.UnmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				utils.Log("ShareId", target.ShareId)
				utils.Log("ShareLink", target.ShareLink)
				utils.Log("SharePassword", target.SharePassword)
			} else {
				utils.ErrorLog("share file failed:", target.Result.Msg)
			}
			putData(target.ReqId, HTTPShareFile, &target)
		} else {
			peers.TransferSendMessageToPPServByP2pAddress(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}
}

// ReqDeleteShare
func ReqDeleteShare(ctx context.Context, conn core.WriteCloser) {
	// pp send to Index Node
	peers.TransferSendMessageToIndexNodeServer(core.MessageFromContext(ctx))
}

// RspDeleteShare
func RspDeleteShare(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspDeleteShare
	if requests.UnmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				utils.Log("cancel share success:", target.ShareId)
			} else {
				utils.ErrorLog("cancel share failed:", target.Result.Msg)
			}
			putData(target.ReqId, HTTPDeleteShare, &target)
		} else {
			peers.TransferSendMessageToPPServByP2pAddress(target.P2PAddress, core.MessageFromContext(ctx))
		}
	}

}

// GetShareFile
func GetShareFile(keyword, sharePassword, saveAs, reqID string, w http.ResponseWriter) {
	utils.DebugLog("GetShareFile for file ", keyword)
	if setting.CheckLogin() {
		peers.SendMessageDirectToIndexNodeOrViaPP(requests.ReqGetShareFileData(keyword, sharePassword, saveAs, reqID), header.ReqGetShareFile)
		storeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqGetShareFile
func ReqGetShareFile(ctx context.Context, conn core.WriteCloser) {
	// pp send to Index Node
	utils.DebugLog("ReqGetShareFile: transferring message to SP server")
	peers.TransferSendMessageToIndexNodeServer(core.MessageFromContext(ctx))
}

// RspGetShareFile
func RspGetShareFile(ctx context.Context, _ core.WriteCloser) {
	var target protos.RspGetShareFile
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	if target.ShareRequest.P2PAddress != setting.P2PAddress {
		peers.TransferSendMessageToPPServByP2pAddress(target.ShareRequest.P2PAddress, core.MessageFromContext(ctx))
		return
	}

	utils.Log("get RspGetShareFile", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		return
	}

	utils.Log("FileInfo:", target.FileInfo)
	putData(target.ShareRequest.ReqId, HTTPGetShareFile, &target)

	for idx, fileInfo := range target.FileInfo {
		saveAs := ""
		if idx == 0 {
			saveAs = target.ShareRequest.SaveAs
		}
		filePath := datamesh.DataMashId{
			Owner: fileInfo.OwnerWalletAddress,
			Hash:  fileInfo.FileHash,
		}.String()
		peers.SendMessageDirectToIndexNodeOrViaPP(requests.ReqFileStorageInfoData(filePath, "", "", setting.WalletAddress,
			saveAs, false, target.ShareRequest), header.ReqFileStorageInfo)
	}
}
