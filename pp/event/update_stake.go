package event

import (
	"context"

	"github.com/stratosnet/framework/core"
	"github.com/stratosnet/framework/utils"
	"github.com/stratosnet/sds-api/header"
	"github.com/stratosnet/sds-api/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/tx"
	"github.com/stratosnet/sds/pp/types"
	txclienttypes "github.com/stratosnet/tx-client/types"
)

// UpdateDeposit Update deposit of node
func UpdateDeposit(ctx context.Context, depositDelta txclienttypes.Coin, txFee txclienttypes.TxFee) error {
	updateDepositReq, err := reqUpdateDepositData(ctx, depositDelta, txFee)
	if err != nil {
		pp.ErrorLog(ctx, "Couldn't build update PP deposit request: "+err.Error())
		return err
	}
	pp.Log(ctx, "Sending update deposit message to SP! "+updateDepositReq.P2PAddress)
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, updateDepositReq, header.ReqUpdateDepositPP)
	return nil
}

// RspUpdateDeposit Response to asking the SP node to update deposit this node
func RspUpdateDeposit(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspUpdateDepositPP
	if err := VerifyMessage(ctx, header.RspUpdateDepositPP, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	success := requests.UnmarshalData(ctx, &target)
	if !success {
		return
	}

	pp.Log(ctx, "get RspUpdateDepositPP", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		return
	}
	setting.State = target.UpdateState

	if target.UpdateState != types.PP_ACTIVE {
		pp.Log(ctx, "Current node isn't activated now")
		return
	}

	err := tx.BroadcastTx(target.Tx)
	if err != nil {
		pp.ErrorLog(ctx, "The UpdateDeposit transaction couldn't be broadcast", err)
	} else {
		pp.Log(ctx, "The UpdateDeposit transaction was broadcast")
	}
}

// NoticeUpdatedDeposit Notice when this PP node's deposit was successfully updated
func NoticeUpdatedDeposit(ctx context.Context, conn core.WriteCloser) {
	var target protos.NoticeUpdatedDepositPP
	if err := VerifyMessage(ctx, header.NoticeUpdatedDepositPP, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	success := requests.UnmarshalData(ctx, &target)
	if !success {
		return
	}
	utils.Logf("get NoticeUpdatedDepositPP, DepositBalance: %v, NodeTier: %v, Weight_Score: %v", target.DepositBalance, target.NodeTier, target.WeightScore)

	// msg is not empty after deposit being updated to 0wei
	depositBalanceAfter, err := txclienttypes.ParseCoinNormalized(target.DepositBalance)
	if err != nil {
		return
	}
	if len(target.Result.Msg) > 0 &&
		depositBalanceAfter.IsZero() &&
		target.NodeTier == "0" {
		// change pp state to unbonding
		setting.State = types.PP_UNBONDING
		pp.Log(ctx, "All tokens are being unbonded(taking around 180 days to complete)"+
			"\n --- This node will be forced to suspend very soon! ---")
	}
	utils.Log("Waiting for state change to be completed")
}
