package server

import (
	"context"

	"github.com/stratosnet/sds/framework/utils"

	"github.com/stratosnet/sds/relayer/cmd/relayd/setting"
	"github.com/stratosnet/sds/relayer/namespace"
	"github.com/stratosnet/sds/relayer/rpc"
	"github.com/stratosnet/sds/relayer/utils/environment"
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
			Namespace: "relayer",
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
