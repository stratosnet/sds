package event

import (
	"context"

	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/relay/stratoschain/grpc"
	utiltypes "github.com/stratosnet/sds/utils/types"
)

// Broadcast send tx to stratos-chain directly
func Send(ctx context.Context, amount utiltypes.Coin, toAddr []byte, txFee utiltypes.TxFee) error {
	sendTxBytes, err := reqSendData(ctx, amount, toAddr, txFee)
	if err != nil {
		pp.ErrorLog(ctx, "Couldn't build send transaction: "+err.Error())
		return err
	}

	err = grpc.BroadcastTx(sendTxBytes, sdktx.BroadcastMode_BROADCAST_MODE_BLOCK)
	if err != nil {
		pp.ErrorLog(ctx, "The send transaction couldn't be broadcast", err)
		return err
	}

	reqId := core.GetRemoteReqId(ctx)
	if reqId != "" {
		rpcResult := &rpc.SendResult{
			Return: rpc.SUCCESS,
		}
		defer pp.SetSendResult(setting.WalletAddress+reqId, rpcResult)
	}

	pp.Log(ctx, "Send transaction delivered.")
	return nil
}
