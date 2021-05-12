package stratoschain

import (
	"errors"
	tmhttp "github.com/tendermint/tendermint/rpc/client/http"
)

func DialWebsocket(addr string) (*tmhttp.HTTP, error) {
	client, err := tmhttp.New("tcp://"+addr, "/websocket")
	if err != nil {
		return nil, errors.New("failed to create stratos-chain Client: " + err.Error())
	}
	err = client.Start()
	if err != nil {
		return nil, errors.New("failed to start stratos-chain Client: " + err.Error())
	}

	return client, nil
}
