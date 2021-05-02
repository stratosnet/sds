package stratoschain

import (
	"context"
	"errors"
	tmhttp "github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

func DialWebsocket(addr, query string) (*tmhttp.HTTP, <-chan coretypes.ResultEvent, error) {
	client, err := tmhttp.New("tcp://"+addr, "/websocket")
	if err != nil {
		return nil, nil, errors.New("failed to create stratos-chain client: " + err.Error())
	}
	err = client.Start()
	if err != nil {
		return nil, nil, errors.New("failed to start stratos-chain client: " + err.Error())
	}

	out, err := client.Subscribe(context.Background(), "relayd", query)
	if err != nil {
		return nil, nil, errors.New("failed to subscribe to query in stratos-chain: " + err.Error())
	}

	return client, out, nil
}
