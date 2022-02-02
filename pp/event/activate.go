package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/relay/stratoschain"
	"github.com/stratosnet/sds/utils"
)

// Activate Inactive PP node becomes active
func Activate(amount, fee, gas int64) error {
	// Query blockchain to know if this node is already a resource node
	ppState, err := stratoschain.QueryResourceNodeState(setting.GetNetworkID().String())
	if err != nil {
		utils.ErrorLog("Couldn't query node status from the blockchain", err)
		return err
	}

	var activateReq *protos.ReqActivatePP
	switch ppState {
	case types.PP_ACTIVE:
		utils.Log("This node is already active on the blockchain. Waiting for SP node to confirm...")
		activateReq = &protos.ReqActivatePP{
			PpInfo:        setting.GetPPInfo(),
			AlreadyActive: true,
		}
	default:
		activateReq, err = reqActivateData(amount, fee, gas)
		if err != nil {
			utils.ErrorLog("Couldn't build PP activate request", err)
			return err
		}
	}

	utils.Log("Sending activate message to SP! " + activateReq.PpInfo.P2PAddress)
	peers.SendMessageToSPServer(activateReq, header.ReqActivatePP)
	return nil
}

// RspActivate. Response to asking the SP node to activate this PP node
func RspActivate(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspActivatePP
	success := requests.UnmarshalData(ctx, &target)
	if !success {
		return
	}

	utils.Log("get RspActivatePP", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		return
	}

	if target.ActivationState != types.PP_INACTIVE {
		utils.Log("Current node is already active")
		setting.State = byte(target.ActivationState)
		return
	}

	err := stratoschain.BroadcastTxBytes(target.Tx)
	if err != nil {
		utils.ErrorLog("The activation transaction couldn't be broadcast", err)
	} else {
		utils.Log("The activation transaction was broadcast")
	}
}

// RspActivated. Response when this PP node was successfully activated
func RspActivated(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspActivatePP
	success := requests.UnmarshalData(ctx, &target)
	if !success {
		return
	}
	utils.Log("get RspActivatedPP", target.Result.State, target.Result.Msg)

	setting.State = types.PP_ACTIVE
	utils.Log("This PP node is now active")
}
