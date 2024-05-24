package stratoschain

import (
	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"

	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/tx-client/grpc"

	"github.com/stratosnet/sds/relayer/cmd/relayd/setting"
	"github.com/stratosnet/sds/relayer/stratoschain/handlers"
)

func BroadcastTx(txBytes []byte) error {

	resp, err := grpc.BroadcastTx(txBytes, txv1beta1.BroadcastMode_BROADCAST_MODE_SYNC)
	if err != nil {
		return err
	}

	if setting.Config == nil {
		return nil // If the relayd config is nil, then this is ppd broadcasting a tx. We don't want to call the event handler in this case
	}

	if len(resp.TxResponse.Logs) == 0 {
		return nil
	}

	events := handlers.ExtractEventsFromTxResponse(resp.TxResponse)
	for _, event := range events {
		msgType := handlers.GetMsgType(event)
		if handler, ok := handlers.Handlers[msgType]; ok {
			go handler(event)
		} else {
			utils.ErrorLogf("No handler for event type [%v]", msgType)
		}
	}
	return nil
}
