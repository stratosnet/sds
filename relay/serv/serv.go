package serv

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/utils/environment"

	"github.com/stratosnet/sds/cmd/relayd/setting"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/namespace"
	"github.com/stratosnet/sds/rpc"
	"github.com/stratosnet/sds/utils"
)

const (
	Home              string = "home"
	SpHome            string = "sp-home"
	Config            string = "config"
	DefaultConfigPath string = "./config/config.toml"
)

var BaseServer = &BaseRelayServer{}

// BaseServer base pp server
type BaseRelayServer struct {
	ppNetwork   *network.Network
	ipcServ     *namespace.IpcServer
	httpRpcServ *namespace.HttpServer
}

func (bs *BaseRelayServer) Start() error {
	utils.Logf("initializing resource node with environment=%v...", environment.GetEnvironment())

	err := bs.startIPC()
	if err != nil {
		return err
	}

	err = bs.startHttpRPC()
	if err != nil {
		return err
	}
	return nil
}

func (bs *BaseRelayServer) startIPC() error {
	rpcAPIs := []rpc.API{
		{
			Namespace: "relayrpc",
			Version:   "1.0",
			Service:   namespace.RpcPrivApiRelay(),
			Public:    false,
		},
	}

	ipc := namespace.NewIPCServer(setting.IpcEndpoint)
	if err := ipc.Start(rpcAPIs, context.Background()); err != nil {
		return err
	}
	bs.ipcServ = ipc

	return nil
}

func (bs *BaseRelayServer) startHttpRPC() error {
	file.RpcWaitTimeout = rpc.DefaultHTTPTimeouts.IdleTimeout
	rpcServer := namespace.NewHTTPServer(rpc.DefaultHTTPTimeouts)
	port, err := strconv.Atoi(setting.Config.Node.Connectivity.RpcPort)
	if err != nil {
		return err
	}

	if err := rpcServer.SetListenAddr("0.0.0.0", port); err != nil {
		return err
	}

	allowModuleList := []string{"user"}
	// if config
	if setting.Config.Node.Connectivity.AllowOwnerRpc {
		allowModuleList = append(allowModuleList, "owner")
	}

	var config = namespace.HttpConfig{
		CorsAllowedOrigins: []string{""},
		Vhosts:             []string{"localhost"},
		Modules:            allowModuleList,
	}

	if err := rpcServer.EnableRPC(namespace.RelayApis(), config); err != nil {
		return err
	}
	if err := rpcServer.Start(context.Background()); err != nil {
		return err
	}

	bs.httpRpcServ = rpcServer
	return nil
}

func (bs *BaseRelayServer) Stop() {
	utils.DebugLogf("BaseServer.Stop ... ")
	if bs.ipcServ != nil {
		_ = bs.ipcServ.Stop()
	}
	if bs.httpRpcServ != nil {
		bs.httpRpcServ.Stop()
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
