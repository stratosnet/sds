package stratoschain

import (
	"errors"
	"os"

	tmlog "github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/rpc/client/http"
	"github.com/stratosnet/sds/framework/utils"
)

func DialWebsocket(addr string) (*http.HTTP, error) {
	url, err := utils.ParseUrl(addr)
	if err != nil {
		return nil, err
	}
	var Logger tmlog.Logger
	Logger = tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout))
	Logger.With("module", "something")

	client, err := http.New(url.String(true, true, false, false), "/websocket")
	if err != nil {
		return nil, errors.New("failed to create stratos-chain Client: " + err.Error())
	}
	client.SetLogger(Logger)
	err = client.Start()
	if err != nil {
		return nil, errors.New("failed to start stratos-chain Client: " + err.Error())
	}

	return client, nil
}
