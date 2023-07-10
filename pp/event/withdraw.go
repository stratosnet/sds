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

// Broadcast withdraw tx to stratos-chain directly
func Withdraw(ctx context.Context, amount utiltypes.Coin, targetAddr []byte, txFee utiltypes.TxFee) error {
	withdrawTxBytes, err := reqWithdrawData(ctx, amount, targetAddr, txFee)
	if err != nil {
		pp.ErrorLog(ctx, "Couldn't build withdraw transaction: "+err.Error())
		return err
	}

	err = grpc.BroadcastTx(withdrawTxBytes, sdktx.BroadcastMode_BROADCAST_MODE_BLOCK)
	if err != nil {
		pp.ErrorLog(ctx, "The withdraw transaction couldn't be broadcast", err)
		return err
	}

	reqId := core.GetRemoteReqId(ctx)
	if reqId != "" {
		rpcResult := &rpc.WithdrawResult{
			Return: rpc.SUCCESS,
		}
		defer pp.SetWithdrawResult(setting.WalletAddress+reqId, rpcResult)
	}

	pp.Log(ctx, "Withdraw transaction delivered.")
	return nil
}
