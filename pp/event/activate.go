package event

import (
	"context"
	"fmt"

	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/relay/stratoschain/grpc"
	"github.com/stratosnet/sds/utils"
	utiltypes "github.com/stratosnet/sds/utils/types"
)

// Activate Inactive PP node becomes active
func Activate(ctx context.Context, amount utiltypes.Coin, txFee utiltypes.TxFee) error {
	// Query blockchain to know if this node is already a resource node
	ppState, _ := grpc.QueryResourceNodeState(p2pserver.GetP2pServer(ctx).GetP2PAddress())

	var activateReq *protos.ReqActivatePP
	var err error
	switch ppState.IsActive {
	case types.PP_ACTIVE:
		pp.Log(ctx, "This node is already active on the blockchain. Waiting for SP node to confirm...")
		activateReq = &protos.ReqActivatePP{
			PpInfo:        p2pserver.GetP2pServer(ctx).GetPPInfo(),
			AlreadyActive: true,
		}
	default:
		activateReq, err = reqActivateData(ctx, amount, txFee)
		if err != nil {
			pp.ErrorLog(ctx, "Couldn't build PP activate request", err)
			return err
		}
	}
	var logstring string
	if p2pserver.GetP2pServer(ctx).SpConnValid() {
		logstring = fmt.Sprintf("Sending activate message to SP: %s, from: %s", p2pserver.GetP2pServer(ctx).GetSpName(), activateReq.PpInfo.P2PAddress)
	} else {
		logstring = fmt.Sprintf("Sending activate message to SP: %s, from: %s", "[no connected sp]", activateReq.PpInfo.P2PAddress)
	}
	utils.Log(logstring)
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, activateReq, header.ReqActivatePP)

	return nil
}

// RspActivate Response to asking the SP node to activate this PP node
func RspActivate(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspActivatePP
	if err := VerifyMessage(ctx, header.RspActivatePP, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}

	success := requests.UnmarshalData(ctx, &target)
	if !success {
		return
	}

	rpcResult := &rpc.ActivateResult{}
	reqId := core.GetRemoteReqId(ctx)
	if reqId != "" {
		defer pp.SetActivateResult(setting.WalletAddress+reqId, rpcResult)
	}

	pp.Log(ctx, "get RspActivatePP", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		rpcResult.Return = rpc.INTERNAL_COMM_FAILURE
		return
	}

	rpcResult.ActivationState = target.ActivationState
	if target.ActivationState == types.PP_ACTIVE {
		pp.Log(ctx, "Current node is already active")
		setting.State = target.ActivationState
		return
	}

	switch target.ActivationState {
	case types.PP_INACTIVE:
		err := grpc.BroadcastTx(target.Tx, sdktx.BroadcastMode_BROADCAST_MODE_BLOCK)
		if err != nil {
			pp.ErrorLog(ctx, "The activation transaction couldn't be broadcast", err)
		} else {
			pp.Log(ctx, "The activation transaction was broadcast")
		}
	case types.PP_ACTIVE:
		pp.Log(ctx, "This node is already active")
	case types.PP_UNBONDING:
		pp.Log(ctx, "This node is unbonding")
	}
	rpcResult.Return = rpc.SUCCESS
}

// NoticeActivatedPP Notice when this PP node was successfully activated
func NoticeActivatedPP(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspActivatePP
	success := requests.UnmarshalData(ctx, &target)
	if !success {
		return
	}
	utils.Log("get NoticeActivatedPP", target.Result.State, target.Result.Msg)

	setting.State = types.PP_ACTIVE
	network.GetPeer(ctx).RunFsm(ctx, network.EVENT_RCV_RSP_ACTIVATED)
	utils.Log("This PP node is now active")

	ReqStateChange(ctx, conn)
}
