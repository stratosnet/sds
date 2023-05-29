package event

import (
	"context"

	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/relay/stratoschain/grpc"
	"github.com/stratosnet/sds/utils"
	utiltypes "github.com/stratosnet/sds/utils/types"
)

// Deactivate Request that an active PP node becomes inactive
func Deactivate(ctx context.Context, txFee utiltypes.TxFee) error {
	deactivateReq, err := reqDeactivateData(ctx, txFee)
	if err != nil {
		pp.ErrorLog(ctx, "Couldn't build PP deactivate request: "+err.Error())
		return err
	}
	pp.Log(ctx, "Sending deactivate message to SP! "+deactivateReq.P2PAddress)
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, deactivateReq, header.ReqDeactivatePP)
	return nil
}

// RspDeactivate Response to asking the SP node to deactivate this PP node
func RspDeactivate(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspDeactivatePP
	if err := VerifyMessage(ctx, header.RspDeactivatePP, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}

	success := requests.UnmarshalData(ctx, &target)
	if !success {
		return
	}

	pp.Log(ctx, "get RspDeactivatePP", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		return
	}

	setting.State = target.ActivationState

	if target.ActivationState == types.PP_INACTIVE {
		pp.Log(ctx, "Current node is already inactive")
		return
	}

	err := grpc.BroadcastTx(target.Tx, sdktx.BroadcastMode_BROADCAST_MODE_BLOCK)
	if err != nil {
		pp.ErrorLog(ctx, "The deactivation transaction couldn't be broadcast", err)
	} else {
		pp.Log(ctx, "The deactivation transaction was broadcast")
	}
}

// RspUnbondingPP. Response when this PP node unbonded all its stake
func RspUnbondingPP(ctx context.Context, conn core.WriteCloser) {
	setting.State = types.PP_UNBONDING
	pp.Log(ctx, "Deactivation submitted, all tokens are being unbonded(taking around 180 days to complete)"+
		"\n --- This node will be forced to suspend very soon! ---")
}

// RspDeactivated. Response when this PP node was successfully deactivated after threshold period (180 days)
func RspDeactivated(ctx context.Context, conn core.WriteCloser) {
	setting.State = types.PP_INACTIVE
	pp.Log(ctx, "This PP node is now deactivated")
}
