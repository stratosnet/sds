package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/relay/stratoschain"
	"github.com/stratosnet/sds/utils"
	utiltypes "github.com/stratosnet/sds/utils/types"
)

// Update stake of node
func UpdateStake(ctx context.Context, stakeDelta utiltypes.Coin, fee utiltypes.Coin, gas int64, incrStake bool) error {
	updateStakeReq, err := reqUpdateStakeData(stakeDelta, fee, gas, incrStake)
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

	err := stratoschain.BroadcastTxBytes(target.Tx)
	if err != nil {
		pp.ErrorLog(ctx, "The UpdateStake transaction couldn't be broadcast", err)
	} else {
		pp.Log(ctx, "The UpdateStake transaction was broadcast")
	}
}

// RspUpdatedStake Response when this PP node's stake was successfully updated
func RspUpdatedStake(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspUpdatedStakePP
	success := requests.UnmarshalData(ctx, &target)
	if !success {
		return
	}
	utils.Logf("get RspUpdatedStakePP, StakeBalance: %v, NodeTier: %v, Weight_Score: %v", target.StakeBalance, target.NodeTier, target.WeightScore)
}
