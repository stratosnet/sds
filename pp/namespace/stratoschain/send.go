package stratoschain

import (
	"context"

	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/relay/stratoschain"
	"github.com/stratosnet/sds/relay/stratoschain/grpc"
	relaytypes "github.com/stratosnet/sds/relay/types"
	"github.com/stratosnet/sds/utils/types"
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

func reqSendData(_ context.Context, amount types.Coin, toAddr []byte, txFee types.TxFee) ([]byte, error) {
	senderAddress, err := types.WalletAddressFromBech(setting.WalletAddress)
	if err != nil {
		return nil, err
	}

	txMsg := stratoschain.BuildSendMsg(senderAddress.Bytes(), toAddr, amount)
	signatureKeys := []relaytypes.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey, Type: relaytypes.SignatureSecp256k1},
	}

	txBytes, err := event.CreateAndSimulateTx(txMsg, banktypes.TypeMsgSend, txFee, "", signatureKeys)
	if err != nil {
		return nil, err
	}

	return txBytes, nil
}
