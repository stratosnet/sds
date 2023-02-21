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
)

// GetWalletOz queries current ozone balance
func GetWalletOz(ctx context.Context, walletAddr, reqId string) error {
	pp.Logf(ctx, "Querying current ozone balance of the wallet: %v", walletAddr)
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqGetWalletOzData(walletAddr, reqId), header.ReqGetWalletOz)
	return nil
}

func RspGetWalletOz(ctx context.Context, conn core.WriteCloser) {
	pp.DebugLog(ctx, "get GetWalletOz RSP")
	var target protos.RspGetWalletOz
	if !requests.UnmarshalData(ctx, &target) {
		pp.DebugLog(ctx, "Cannot unmarshal ozone balance data")
		return
	}

	rpcResult := &rpc.GetOzoneResult{}
	reqId := core.GetRemoteReqId(ctx)
	if reqId != "" {
		defer file.SetQueryOzoneResult(target.WalletAddress+reqId, rpcResult)
	}

	if target.Result.State != protos.ResultState_RES_SUCCESS {
		pp.Logf(ctx, "failed to get ozone balance: %v", target.Result.Msg)
		rpcResult.Return = rpc.INTERNAL_COMM_FAILURE
		return
	}
	pp.Logf(ctx, "get GetWalletOz RSP, the current ozone balance of %v = %v, %v", target.GetWalletAddress(), target.GetWalletOz(), reqId)
	rpcResult.Return = rpc.SUCCESS
	rpcResult.Ozone = target.WalletOz
}
