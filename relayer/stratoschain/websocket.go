package stratoschain

import (
	"errors"

	"github.com/cometbft/cometbft/rpc/client/http"
	"github.com/stratosnet/sds/framework/utils"
)

func DialWebsocket(addr string) (*http.HTTP, error) {
	url, err := utils.ParseUrl(addr)
	if err != nil {
		return nil, err
	}

	client, err := http.New(url.String(true, true, false, false), "/websocket")
	if err != nil {
		return nil, errors.New("failed to create stratos-chain Client: " + err.Error())
	}
	err = client.Start()
	if err != nil {
		return nil, errors.New("failed to start stratos-chain Client: " + err.Error())
	}

	return client, nil
}
