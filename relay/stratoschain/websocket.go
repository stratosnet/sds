package stratoschain

import (
	"errors"

	"github.com/stratosnet/sds/utils"
	tmhttp "github.com/tendermint/tendermint/rpc/client/http"
)

func DialWebsocket(addr string) (*tmhttp.HTTP, error) {
	url, err := utils.ParseUrl(addr)
	if err != nil {
		return nil, err
	}

	client, err := tmhttp.New(url.String(true, true, false, false), "/websocket")
	if err != nil {
		return nil, errors.New("failed to create stratos-chain Client: " + err.Error())
	}
	err = client.Start()
	if err != nil {
		return nil, errors.New("failed to start stratos-chain Client: " + err.Error())
	}

	return client, nil
}
