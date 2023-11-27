package tx

import (
	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"
	"github.com/stratosnet/tx-client/grpc"
)

func BroadcastTx(txBytes []byte) error {
	_, err := grpc.BroadcastTx(txBytes, txv1beta1.BroadcastMode_BROADCAST_MODE_SYNC)
	if err != nil {
		return err
	}
	// pp will not call the event handler after broadcasting a tx.
	return nil
}
