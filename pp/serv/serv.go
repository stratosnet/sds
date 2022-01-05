package serv

import (
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/rpc"
	"github.com/stratosnet/sds/utils"
)

func Start() {
	err := GetWalletAddress()
	if err != nil {
		utils.ErrorLog(err)
		return
	}

	err = startIPC()

	if err != nil {
		utils.ErrorLog(err)
		return
	}

	peers.StartPP(event.RegisterEventHandle)
}

func startIPC() error {
	rpcAPIs := []rpc.API{
		{
			Namespace: "sds",
			Version:   "1.0",
			Service:   TerminalAPI(),
			Public:    false,
		},
		{
			Namespace: "sdslog",
			Version:   "1.0",
			Service:   RpcLogService(),
			Public:    false,
		},
	}

	ipc := newIPCServer(setting.IpcEndpoint)
	if err := ipc.start(rpcAPIs); err != nil {
		return err
	}

	//TODO bring this back later once we have a proper quit mechanism
	//defer ipc.stop()

	return nil
}
