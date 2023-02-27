package grpc

import (
	"context"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/stratosnet/sds/cmd/relayd/setting"
	"github.com/stratosnet/sds/relay/stratoschain/handlers"
	"github.com/stratosnet/sds/relay/stratoschain/tx"
	"github.com/stratosnet/sds/utils"
)

func BroadcastTx(txBytes []byte, mode sdktx.BroadcastMode) error {
	conn, err := CreateGrpcConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := sdktx.NewServiceClient(conn)
	ctx := context.Background()
	req := sdktx.BroadcastTxRequest{TxBytes: txBytes, Mode: mode}

	resp, err := client.BroadcastTx(ctx, &req)
	if err != nil {
		return err
	}

	if setting.Config == nil {
		return nil // If the relayd config is nil, then this is ppd broadcasting a tx. We don't want to call the event handler in this case
	}
	events := tx.ProcessEvents(*resp)
	for msgType, event := range events {
		if handler, ok := handlers.Handlers[msgType]; ok {
			go handler(event)
		} else {
			utils.ErrorLogf("No handler for event type [%v]", msgType)
		}
	}
	return nil
}

func Simulate(txBytes []byte) (*sdktypes.GasInfo, error) {
	conn, err := CreateGrpcConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := sdktx.NewServiceClient(conn)
	ctx := context.Background()
	req := sdktx.SimulateRequest{TxBytes: txBytes}

	resp, err := client.Simulate(ctx, &req)
	if err != nil {
		return nil, err
	}

	return resp.GetGasInfo(), nil
}
