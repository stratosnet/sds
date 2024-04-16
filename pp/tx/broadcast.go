package tx

import (
	"errors"

	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"
	"github.com/stratosnet/sds/tx-client/grpc"
)

func BroadcastTx(txBytes []byte) error {
	rsp, err := grpc.BroadcastTx(txBytes, txv1beta1.BroadcastMode_BROADCAST_MODE_SYNC)
	if err != nil {
		return err
	}
	if rsp.GetTxResponse().Code != 0 {
		return errors.New("tx failed")
	}
	// pp will not call the event handler after broadcasting a tx.
	return nil
}
