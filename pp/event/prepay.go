package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/msg/header"
	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/tx"
	"github.com/stratosnet/sds/sds-msg/protos"
	txclienttypes "github.com/stratosnet/sds/tx-client/types"
)

// Prepay PP node sends a prepay transaction
func Prepay(ctx context.Context, beneficiary fwtypes.WalletAddress, amount txclienttypes.Coin, txFee txclienttypes.TxFee,
	walletAddr string, walletPubkey, wsign []byte, reqTime int64) error {
	prepayReq, err := reqPrepayData(ctx, beneficiary, amount, txFee, walletAddr, walletPubkey, wsign, reqTime)
	if err != nil {
		pp.ErrorLog(ctx, "Couldn't build PP prepay request: "+err.Error())
		return err
	}
	pp.Log(ctx, "Sending prepay message to SP! "+prepayReq.Signature.Address)
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, prepayReq, header.ReqPrepay)
	return nil
}

// RspPrepay Response to asking the SP node to send a prepay transaction
func RspPrepay(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspPrepay
	if err := VerifyMessage(ctx, header.RspPrepay, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	success := requests.UnmarshalData(ctx, &target)
	if !success {
		return
	}
	rpcResult := &rpc.PrepayResult{}
	reqId := core.GetRemoteReqId(ctx)
	if reqId != "" {
		defer pp.SetRPCResult(setting.WalletAddress+reqId, rpcResult)
	}
	pp.Log(ctx, "get RspPrepay", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		rpcResult.Return = rpc.INTERNAL_COMM_FAILURE
		return
	}

	err := tx.BroadcastTx(target.Tx)
	if err != nil {
		pp.ErrorLog(ctx, "The prepay transaction couldn't be broadcast", err)
	} else {
		pp.Log(ctx, "The prepay transaction was broadcast")
	}
	rpcResult.Return = rpc.SUCCESS
}
