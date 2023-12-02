package stratoschain

import (
	"context"

	"github.com/cosmos/cosmos-proto/anyutil"

	"github.com/stratosnet/sds/framework/core"
	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/tx"
	txclienttx "github.com/stratosnet/sds/tx-client/tx"
	txclienttypes "github.com/stratosnet/sds/tx-client/types"
)

// Broadcast send tx to stratos-chain directly
func Send(ctx context.Context, amount txclienttypes.Coin, toAddr fwtypes.WalletAddress, txFee txclienttypes.TxFee) error {
	sendTxBytes, err := reqSendData(ctx, amount, toAddr, txFee)
	if err != nil {
		pp.ErrorLog(ctx, "Couldn't build send transaction: "+err.Error())
		return err
	}

	err = tx.BroadcastTx(sendTxBytes)
	if err != nil {
		pp.ErrorLog(ctx, "The send transaction couldn't be broadcast", err)
		return err
	}

	reqId := core.GetRemoteReqId(ctx)
	if reqId != "" {
		rpcResult := &rpc.SendResult{
			Return: rpc.SUCCESS,
		}
		defer pp.SetRPCResult(setting.WalletAddress+reqId, rpcResult)
	}

	pp.Log(ctx, "Send transaction delivered.")
	return nil
}

func reqSendData(_ context.Context, amount txclienttypes.Coin, toAddr fwtypes.WalletAddress, txFee txclienttypes.TxFee) ([]byte, error) {
	senderAddress, err := fwtypes.WalletAddressFromBech32(setting.WalletAddress)
	if err != nil {
		return nil, err
	}

	txMsg := txclienttx.BuildSendMsg(senderAddress, toAddr, amount)
	signatureKeys := []*txclienttypes.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey.Bytes(), Type: txclienttypes.SignatureSecp256k1},
	}

	chainId := setting.Config.Blockchain.ChainId
	gasAdjustment := setting.Config.Blockchain.GasAdjustment

	msgAny, err := anyutil.New(txMsg)
	if err != nil {
		return nil, err
	}

	txBytes, err := txclienttx.CreateAndSimulateTx(msgAny, txFee, "", signatureKeys, chainId, gasAdjustment)
	if err != nil {
		return nil, err
	}

	return txBytes, nil
}
