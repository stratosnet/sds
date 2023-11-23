package server

import (
	"context"
	"fmt"

	"github.com/stratosnet/tx-client/grpc"

	"github.com/stratosnet/framework/utils"

	"github.com/stratosnet/relay/stratoschain/handlers"
)

const (
	DefaultMsg = "Request Accepted"
)

type CmdResult struct {
	Msg string
}

type relayCmd struct {
}

func RelayAPI() *relayCmd {
	return &relayCmd{}
}

func (api *relayCmd) Sync(ctx context.Context, param []string) (CmdResult, error) {
	if len(param) != 1 || len(param[0]) == 0 {
		utils.ErrorLog("wrong number of arguments")
		return CmdResult{Msg: ""}, fmt.Errorf("wrong number of arguments")
	}
	txHash := param[0]
	txResponse, err := grpc.QueryTxByHash(txHash)
	if err != nil {
		errMsg := fmt.Sprintf("error when calling grpc.QueryTxByHash for txHash[%v], reason: %v", txHash, err.Error())
		utils.DebugLogf(errMsg)
		return CmdResult{Msg: ""}, fmt.Errorf(errMsg)
	}

	// process relayed events
	events := handlers.ProcessEvents(txResponse)
	for msgType, event := range events {
		if handler, ok := handlers.Handlers[msgType]; ok {
			go handler(event)
		} else {
			utils.ErrorLogf("No handler for event type [%v]", msgType)
		}
	}

	return CmdResult{Msg: DefaultMsg}, nil
}
