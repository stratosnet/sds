package serv

import (
	"errors"
	"strconv"

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

	err = startMonitor()
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

func startMonitor() error {
	monitorServer := newHTTPServer(rpc.DefaultHTTPTimeouts)

	port, err := strconv.Atoi(setting.Config.MonitorPort)
	if err != nil {
		return errors.New("wrong configuration for monitor port")
	}

	if err := monitorServer.setListenAddr("0.0.0.0", port); err != nil {
		return err
	}

	var config = wsConfig{
		Origins: []string{},
		Modules: []string{},
		prefix:  "",
	}

	if err := monitorServer.enableWS(monitorAPI(), config); err != nil {
		return err
	}

	if err := monitorServer.start(); err != nil {
		return err
	}

	return nil
}
