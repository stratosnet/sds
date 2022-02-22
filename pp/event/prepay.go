package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/relay/stratoschain"
	"github.com/stratosnet/sds/utils"
)

// Prepay PP node sends a prepay transaction
func Prepay(amount, fee, gas int64) error {
	prepayReq, err := reqPrepayData(amount, fee, gas)
	if err != nil {
		utils.ErrorLog("Couldn't build PP prepay request: " + err.Error())
		return err
	}
	utils.Log("Sending prepay message to SP! " + prepayReq.WalletAddress)
	peers.SendMessageToSPServer(prepayReq, header.ReqPrepay)
	return nil
}

// RspPrepay. Response to asking the SP node to send a prepay transaction
func RspPrepay(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspPrepay
	success := requests.UnmarshalData(ctx, &target)
	if !success {
		return
	}

	utils.Log("get RspPrepay", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		return
	}

	err := stratoschain.BroadcastTxBytes(target.Tx)
	if err != nil {
		utils.ErrorLog("The prepay transaction couldn't be broadcast", err)
	} else {
		utils.Log("The prepay transaction was broadcast")
	}
}

// RspPrepaid. Response when this PP node's prepay transaction was successful
func RspPrepaid(ctx context.Context, conn core.WriteCloser) {
	utils.Log("The prepay transaction has been executed")
}
