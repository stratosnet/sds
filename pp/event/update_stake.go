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

// UpdateStake Update stake of node
func UpdateStake(ctx context.Context, stakeDelta utiltypes.Coin, txFee utiltypes.TxFee, incrStake bool) error {
	updateStakeReq, err := reqUpdateStakeData(stakeDelta, txFee, incrStake)
	if err != nil {
		pp.ErrorLog(ctx, "Couldn't build update PP stake request: "+err.Error())
		return err
	}
	pp.Log(ctx, "Sending update stake message to SP! "+updateStakeReq.P2PAddress)
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, updateStakeReq, header.ReqUpdateStakePP)
	return nil
}

// RspUpdateStake Response to asking the SP node to update stake this node
func RspUpdateStake(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspUpdateStakePP
	if err := VerifyMessage(ctx, header.RspUpdatedStakePP, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
	}
	success := requests.UnmarshalData(ctx, &target)
	if !success {
		return
	}

	pp.Log(ctx, "get RspUpdateStakePP", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		return
	}

	if target.UpdateState == types.PP_INACTIVE {
		pp.Log(ctx, "Current node isn't active yet")
	}

	setting.State = target.UpdateState

	err := grpc.BroadcastTx(target.Tx, sdktx.BroadcastMode_BROADCAST_MODE_BLOCK)
	if err != nil {
		pp.ErrorLog(ctx, "The UpdateStake transaction couldn't be broadcast", err)
	} else {
		pp.Log(ctx, "The UpdateStake transaction was broadcast")
	}
}

// RspUpdatedStake Response when this PP node's stake was successfully updated
func RspUpdatedStake(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspUpdatedStakePP
	if err := VerifyMessage(ctx, header.RspUpdatedStakePP, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
	}
	success := requests.UnmarshalData(ctx, &target)
	if !success {
		return
	}
	utils.Logf("get RspUpdatedStakePP, StakeBalance: %v, NodeTier: %v, Weight_Score: %v", target.StakeBalance, target.NodeTier, target.WeightScore)

	// msg is not empty after stake being updated to 0wei
	stakeBalanceAfter, err := utiltypes.ParseCoinNormalized(target.StakeBalance)
	if err != nil {
		return
	}
	if len(target.Result.Msg) > 0 &&
		stakeBalanceAfter.IsZero() &&
		target.NodeTier == "0" {
		// change pp state to unbonding
		setting.State = types.PP_UNBONDING
	}
}
