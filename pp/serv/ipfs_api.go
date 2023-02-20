package serv

import (
	"context"
	nethttp "net/http"

	"github.com/ipfs/go-ipfs-cmds/http"
	"github.com/stratosnet/sds/pp/setting"
)

func StartIpfsServ(ctx context.Context) {
	config := http.NewServerConfig()
	config.APIPath = "/api/v0"
	h := http.NewHandler(CtxEnv{Ctx: ctx}, RootCmd, config)
	port := setting.Config.IpfsRpcPort

	if port == "" {
		return
	}

	// create http rpc server
	err := nethttp.ListenAndServe(":"+port, h)
	if err != nil {
		panic(err)
	}
}
