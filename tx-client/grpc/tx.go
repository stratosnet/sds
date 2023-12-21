package grpc

import (
	"context"

	abciv1beta1 "cosmossdk.io/api/cosmos/base/abci/v1beta1"
	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"

	"github.com/stratosnet/sds/tx-client/utils"
)

func BroadcastTx(txBytes []byte, mode txv1beta1.BroadcastMode) (*txv1beta1.BroadcastTxResponse, error) {
	conn, err := CreateGrpcConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := txv1beta1.NewServiceClient(conn)
	ctx := context.Background()
	req := txv1beta1.BroadcastTxRequest{TxBytes: txBytes, Mode: mode}

	resp, err := client.BroadcastTx(ctx, &req)
	if err != nil {
		return nil, err
	}
	if resp.GetTxResponse().Code != 0 {
		utils.ErrorLogf("Tx failed: [%v]", resp.GetTxResponse().String())
	}
	return resp, nil
}

func Simulate(txBytes []byte) (*abciv1beta1.GasInfo, error) {
	conn, err := CreateGrpcConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := txv1beta1.NewServiceClient(conn)
	ctx := context.Background()
	req := txv1beta1.SimulateRequest{TxBytes: txBytes}

	resp, err := client.Simulate(ctx, &req)
	if err != nil {
		return nil, err
	}

	return resp.GetGasInfo(), nil
}
