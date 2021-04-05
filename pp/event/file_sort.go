package event

import (
	"context"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg/header"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/pp/client"
	"github.com/qsnetwork/sds/pp/setting"
	"github.com/qsnetwork/sds/utils"
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
		if target.WalletAddress == setting.WalletAddress {
			putData(target.ReqId, HTTPFileSort, &target)
		} else {
			transferSendMessageToClient(target.WalletAddress, spbf.MessageFromContext(ctx))
		}

	}
}
