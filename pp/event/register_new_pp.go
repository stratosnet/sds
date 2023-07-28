package event

// Author j
import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// RegisterNewPP P-SP P register to become PP
func RegisterNewPP(ctx context.Context, walletAddr string, walletPubkey, wsig []byte, reqTime int64) {
	if setting.CheckLogin() {
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx,
			requests.ReqRegisterNewPPData(ctx, walletAddr, walletPubkey, wsig, reqTime),
			header.ReqRegisterNewPP)
	}
}

// RspRegisterNewPP  SP-P
func RspRegisterNewPP(ctx context.Context, conn core.WriteCloser) {
	pp.Log(ctx, "get RspRegisterNewPP")
	var target protos.RspRegisterNewPP
	if err := VerifyMessage(ctx, header.RspRegisterNewPP, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	rpcResult := &rpc.RPResult{}
	reqId := core.GetRemoteReqId(ctx)
	if reqId != "" {
		defer pp.SetRPCResult(p2pserver.GetP2pServer(ctx).GetP2PAddress()+setting.WalletAddress+reqId, rpcResult)
	}
	pp.Log(ctx, "get RspRegisterNewPP", target.Result.State, target.Result.Msg)
	rpcResult.AlreadyPp = target.AlreadyPp
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		rpcResult.Return = rpc.INTERNAL_COMM_FAILURE
		if target.AlreadyPp {
			setting.IsPP = true
			setting.IsPPSyncedWithSP = true
		}
		return
	}

	network.GetPeer(ctx).RunFsm(ctx, network.EVENT_RCV_RSP_REGISTER_NEW_PP)
	pp.Log(ctx, "registered as PP successfully, you can deposit by `activate` ")
	setting.IsPP = true
	setting.IsPPSyncedWithSP = true
	rpcResult.Return = rpc.SUCCESS
}
