package serv

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/stratosnet/sds/utils/environment"

	"github.com/stratosnet/sds/cmd/relayd/setting"
	"github.com/stratosnet/sds/pp/namespace"
	"github.com/stratosnet/sds/rpc"
	"github.com/stratosnet/sds/utils"
)

const (
	Home   string = "home"
	SpHome string = "sp-home"
	Config string = "config"
)

var BaseServer = &BaseRelayServer{}

// BaseServer base pp server
type BaseRelayServer struct {
	ipcServ *namespace.IpcServer
}

func (bs *BaseRelayServer) Start() error {
	utils.Logf("initializing resource node with environment=%v...", environment.GetEnvironment())

	err := bs.startIPC()
	if err != nil {
		return err
	}
	return nil
}

func (bs *BaseRelayServer) startIPC() error {
	rpcAPIs := []rpc.API{
		{
			Namespace: "relay",
			Version:   "1.0",
			Service:   RelayAPI(),
			Public:    false,
		},
	}
	utils.DebugLogf("IpcEndpoint is %v", setting.IpcEndpoint)
	ipc := namespace.NewIPCServer(setting.IpcEndpoint)
	if err := ipc.Start(rpcAPIs, context.Background()); err != nil {
		return err
	}
	bs.ipcServ = ipc

	return nil
}

func (bs *BaseRelayServer) Stop() {
	utils.DebugLogf("BaseRelayServer.Stop ... ")
	if bs.ipcServ != nil {
		_ = bs.ipcServ.Stop()
	}
}

func GetQuitChannel() chan os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)

	return quit
}
