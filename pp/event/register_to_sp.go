package event

// Author j
import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/msg/header"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/sds-msg/protos"
)

// RspRegister  PP -> SP, SP -> PP, PP -> P
func RspRegister(ctx context.Context, conn core.WriteCloser) {
	pp.Log(ctx, "get RspRegister", conn)
	var target protos.RspRegister
	if err := VerifyMessage(ctx, header.RspRegister, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	pp.Log(ctx, "target.RspRegister", target.P2PAddress)
	if target.P2PAddress != p2pserver.GetP2pServer(ctx).GetP2PAddress().String() {
		return
	}

	if target.IsSuspended {
		network.GetPeer(ctx).RunFsm(ctx, network.EVENT_RCV_SUSPENDED_STATE)
	}

	if target.Result.State != protos.ResultState_RES_SUCCESS {
		pp.Log(ctx, "Register failed", target.Result.Msg)
		pp.Log(ctx, "startmining will automatically retry in a few minutes, please wait...")
		return
	}

	pp.Log(ctx, "Register successful", target.Result.Msg)
	setting.IsPPSyncedWithSP = true
	pp.DebugLog(ctx, "@@@@@@@@@@@@@@@@@@@@@@@@@@@@", p2pserver.GetP2pServer(ctx).GetConnectionName(conn))
	setting.IsPP = target.IsPP
	if setting.IsPP {
		network.GetPeer(ctx).StartMining(ctx)
	}
}

// RspMining RspMining
func RspMining(ctx context.Context, conn core.WriteCloser) {
	pp.DebugLog(ctx, "get RspMining", conn)
	var target protos.RspMining
	if err := VerifyMessage(ctx, header.RspMining, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	rpcResult := &rpc.StartMiningResult{}
	reqId := core.GetRemoteReqId(ctx)
	if reqId != "" {
		defer pp.SetRPCResult(p2pserver.GetP2pServer(ctx).GetP2PAddress().String()+reqId, rpcResult)
	}
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		network.GetPeer(ctx).RunFsm(ctx, network.EVENT_RCV_MINING_NOT_STARTED)
		pp.Log(ctx, target.Result.Msg)
		rpcResult.Return = rpc.INTERNAL_COMM_FAILURE
		return
	}

	pp.Log(ctx, "start mining")
	if p2pserver.GetP2pServer(ctx).GetP2pServer() == nil {
		go p2pserver.GetP2pServer(ctx).StartListenServer(ctx, setting.GetP2pServerPort())
	}
	pp.DebugLog(ctx, "Start reporting node status to SP")
	// trigger 1 stat report immediately
	network.GetPeer(ctx).ReportNodeStatus(ctx)
	rpcResult.Return = rpc.SUCCESS
}

// NoticeRelocateSp An SP wants this node to switch to a different SP
func NoticeRelocateSp(ctx context.Context, conn core.WriteCloser) {
	var target protos.NoticeRelocateSp
	if err := VerifyMessage(ctx, header.NoticeRelocateSp, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		pp.DebugLog(ctx, "Cannot unmarshal SP relocation notice")
		return
	}
	if target.P2PAddressPp != p2pserver.GetP2pServer(ctx).GetP2PAddress().String() {
		return
	}
	if target.ToSp == nil {
		pp.DebugLog(ctx, "Target SP is missing in NoticeRelocateSp")
		return
	}

	p2pServer := p2pserver.GetP2pServer(ctx)
	pp.Logf(ctx, "Received a notice to switch to SP %v (%v). Current SP is %v", target.ToSp.NetworkAddress, target.ToSp.P2PAddress, p2pServer.GetSpName())
	p2pServer.ConfirmOptSP(ctx, target.ToSp.NetworkAddress)
}
