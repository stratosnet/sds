package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/relay/stratoschain"
	"github.com/stratosnet/sds/utils"
)

// Update stake of node
func UpdateStake(stakeDelta, fee, gas int64, incrStake bool) error {
	updateStakeReq, err := reqUpdateStakeData(stakeDelta, fee, gas, incrStake)
	if err != nil {
		utils.ErrorLog("Couldn't build update PP stake request: " + err.Error())
		return err
	}
	utils.Log("Sending update stake message to SP! " + updateStakeReq.P2PAddress)
	peers.SendMessageToSPServer(updateStakeReq, header.ReqActivatePP)
	return nil
}

// RspUpdateNodeStake. Response to asking the SP node to update stake this node
func RspUpdateStake(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspUpdateStakePP
	success := types.UnmarshalData(ctx, &target)
	if !success {
		return
	}

	utils.Log("get RspUpdateStakePP", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		return
	}

	if target.UpdateState != setting.PP_ACTIVE {
		utils.Log("Current node isn't active yet")
		setting.State = byte(target.UpdateState)
		return
	}

	err := stratoschain.BroadcastTxBytes(target.Tx)
	if err != nil {
		utils.ErrorLog("The UpdateStake transaction couldn't be broadcast", err)
	} else {
		utils.Log("The UpdateStake transaction was broadcast")
	}
}

// RspUpdated. Response when this PP node was successfully activated
func RspUpdated(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspUpdatedStakePP
	success := types.UnmarshalData(ctx, &target)
	if !success {
		return
	}
	utils.Log("get RspUpdatedStakePP", target.Result.State, target.Result.Msg)

	setting.State = setting.PP_ACTIVE
	utils.Log("This PP node is now active")
}
