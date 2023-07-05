package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

func ClearExpiredShareLinks(ctx context.Context, walletAddr string, walletPubkey, wsign []byte, reqTime int64) {
	if setting.CheckLogin() {
		p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, requests.ClearExpiredShareLinksData(
			p2pserver.GetP2pServer(ctx).GetP2PAddress(), walletAddr, walletPubkey, wsign, reqTime), header.ReqClearExpiredShareLinks)
	}
}

func RspClearExpiredShareLinks(ctx context.Context, conn core.WriteCloser) {
	pp.DebugLog(ctx, "get RspClearExpiredShareLinks")
	var target protos.RspClearExpiredShareLinks
	if err := VerifyMessage(ctx, header.RspClearExpiredShareLinks, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	rpcResult := &rpc.ClearExpiredShareLinksResult{}

	// fail to unmarshal data, not able to determine if and which RPC client this is from, let the client timeout
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	// serv the RPC user when the ReqId is not empty
	reqId := core.GetRemoteReqId(ctx)
	if reqId != "" {
		defer file.SetClearExpiredShareLinksResult(target.WalletAddress+reqId, rpcResult)
	}

	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.ErrorLog(target.Result.Msg)
		rpcResult.Return = rpc.INTERNAL_COMM_FAILURE
		return
	}

	if target.Result.State == protos.ResultState_RES_SUCCESS {
		pp.Log(ctx, "ClearExpiredShareLinks success ", target.Result.Msg)
	} else {
		pp.Log(ctx, "ClearExpiredShareLinks failed ", target.Result.Msg)
	}
}
