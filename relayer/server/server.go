package server

import (
	"context"
	"strconv"

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
	ipcServ     *namespace.IpcServer
	httpRpcServ *namespace.HttpServer
}

func (bs *BaseRelayServer) Start() error {
	utils.Logf("initializing relayer with environment=%v...", environment.GetEnvironment())

	err := bs.startIPC()
	if err != nil {
		return err
	}
	return bs.startHttpRPC()
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

func (bs *BaseRelayServer) startHttpRPC() error {
	rpcServer := namespace.NewHTTPServer(rpc.DefaultHTTPTimeouts)
	port, err := strconv.Atoi(setting.Config.Connectivity.RpcPort)
	if err != nil {
		return err
	}
	if err = rpcServer.SetListenAddr("0.0.0.0", port); err != nil {
		return err
	}

	rpcAPIs := []rpc.API{
		{
			Namespace: "query",
			Version:   "1.0",
			Service:   RpcAPI(),
			Public:    false,
		},
	}
	var config = namespace.HttpConfig{
		CorsAllowedOrigins: []string{""},
		Vhosts:             []string{"localhost"},
		Modules:            []string{"query"},
	}

	if err = rpcServer.EnableRPC(rpcAPIs, config); err != nil {
		return err
	}

	if err = rpcServer.Start(context.Background()); err != nil {
		return err
	}

	bs.httpRpcServ = rpcServer
	return nil
}

func (bs *BaseRelayServer) Stop() {
	utils.DebugLogf("BaseRelayServer.Stop ... ")
	if bs.ipcServ != nil {
		_ = bs.ipcServ.Stop()
	}
	if bs.httpRpcServ != nil {
		bs.httpRpcServ.Stop()
	}
}
