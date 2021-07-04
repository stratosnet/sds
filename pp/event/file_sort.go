package event

import (
	"context"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"net/http"
)

// FileSort
func FileSort(files []*protos.FileInfo, reqID, albumID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		sendMessage(client.PPConn, fileSortData(files, reqID, albumID), header.ReqFileSort)
		stroeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqFileSort ReqFileSort
func ReqFileSort(ctx context.Context, conn spbf.WriteCloser) {
	utils.DebugLog("+++++++++++++++++++++++++++++++++++++++++++++++++++")
	transferSendMessageToSPServer(spbf.MessageFromContext(ctx))
}

// RspFileSort
func RspFileSort(ctx context.Context, conn spbf.WriteCloser) {
	utils.DebugLog("get RspFindMyFileList")
	var target protos.RspFileSort
	if unmarshalData(ctx, &target) {
		if target.P2PAddress == setting.P2PAddress {
			putData(target.ReqId, HTTPFileSort, &target)
		} else {
			transferSendMessageToClient(target.P2PAddress, spbf.MessageFromContext(ctx))
		}
	}
}
