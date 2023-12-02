package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/sds-msg/header"
	"github.com/stratosnet/sds/sds-msg/protos"
)

func DeleteFile(ctx context.Context, fileHash string, walletAddr string, walletPubkey, wsign []byte, reqTime int64) {
	if setting.CheckLogin() {
		p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx,
			requests.ReqDeleteFileData(fileHash, p2pserver.GetP2pServer(ctx).GetP2PAddress().String(), walletAddr, walletPubkey, wsign, reqTime),
			header.ReqDeleteFile)
	}
}

func RspDeleteFile(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspDeleteFile
	if err := VerifyMessage(ctx, header.RspDeleteFile, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
	}
	if requests.UnmarshalData(ctx, &target) {
		if target.P2PAddress == p2pserver.GetP2pServer(ctx).GetP2PAddress().String() {
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
