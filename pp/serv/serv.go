package serv

import (
	"context"
	"errors"
	"strconv"

	"github.com/stratosnet/sds/metrics"
	"github.com/stratosnet/sds/pp/account"
	"github.com/stratosnet/sds/pp/api"
	"github.com/stratosnet/sds/pp/api/rest"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/rpc"
	"github.com/stratosnet/sds/utils"
)

// BaseServer base pp server
type BaseServer struct {
	p2pServ     *p2pserver.P2pServer
	ppNetwork   *network.Network
	ipcServ     *ipcServer
	httpRpcServ *httpServer
	monitorServ *httpServer
}

func (bs *BaseServer) Start() error {
	ctx := context.Background()
	err := account.GetWalletAddress(ctx)
	if err != nil {
		return err
	}

	err = bs.startP2pServer()
	if err != nil {
		return err
	}

	err = bs.startInternalApiServer()
	if err != nil {
		return err
	}

	err = bs.startRestServer()
	if err != nil {
		return err
	}

	err = bs.startTrafficLog()
	if err != nil {
		return err
	}

	err = bs.startIPC()
	if err != nil {
		return err
	}

	err = bs.startHttpRPC()
	if err != nil {
		return err
	}

	return bs.startMonitor()
}

func (bs *BaseServer) startIPC() error {
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
	ctx := context.WithValue(context.Background(), types.P2P_SERVER_KEY, bs.p2pServ)
	ctx = context.WithValue(ctx, types.PP_NETWORK_KEY, bs.ppNetwork)
	if err := ipc.start(rpcAPIs, ctx); err != nil {
		return err
	}
	bs.ipcServ = ipc
	//TODO bring this back later once we have a proper quit mechanism
	//defer ipc.stop()

	return nil
}

func (bs *BaseServer) startHttpRPC() error {
	file.RpcWaitTimeout = rpc.DefaultHTTPTimeouts.IdleTimeout
	rpcServer := newHTTPServer(rpc.DefaultHTTPTimeouts)
	port, err := strconv.Atoi(setting.Config.RpcPort)
	if err != nil {
		return err
	}

	if err := rpcServer.setListenAddr("0.0.0.0", port); err != nil {
		return err
	}

	var config = httpConfig{
		CorsAllowedOrigins: []string{""},
		Vhosts:             []string{"localhost"},
		Modules:            nil,
	}

	if err := rpcServer.enableRPC(apis(), config); err != nil {
		return err
	}
	ctx := context.WithValue(context.Background(), types.P2P_SERVER_KEY, bs.p2pServ)
	ctx = context.WithValue(ctx, types.PP_NETWORK_KEY, bs.ppNetwork)
	if err := rpcServer.start(ctx); err != nil {
		return err
	}

	bs.httpRpcServ = rpcServer
	return nil
}

func (bs *BaseServer) startMonitor() error {
	monitorServer := newHTTPServer(rpc.DefaultHTTPTimeouts)
	if setting.Config.Monitor.TLS {
		monitorServer.enableTLS(setting.Config.Monitor.Cert, setting.Config.Monitor.Key)
	}
	port, err := strconv.Atoi(setting.Config.Monitor.Port)
	if err != nil {
		return errors.New("wrong configuration for monitor port")
	}

	_, err = strconv.Atoi(setting.Config.MetricsPort)
	if err != nil {
		return errors.New("wrong configuration for metrics port")
	}

	if err = metrics.Initialize(setting.Config.MetricsPort); err != nil {
		return err
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
	ctx := context.WithValue(context.Background(), types.P2P_SERVER_KEY, bs.p2pServ)
	ctx = context.WithValue(ctx, types.PP_NETWORK_KEY, bs.ppNetwork)
	if err := monitorServer.start(ctx); err != nil {
		return err
	}
	bs.monitorServ = monitorServer
	return nil
}

func (bs *BaseServer) startP2pServer() error {
	bs.p2pServ = &p2pserver.P2pServer{}
	event.RegisterEventHandle()
	ctx := context.Background()
	ctx = context.WithValue(ctx, types.P2P_SERVER_KEY, bs.p2pServ)
	bs.p2pServ.AddConnConntextKey(types.P2P_SERVER_KEY)

	bs.ppNetwork = &network.Network{}
	ctx = context.WithValue(ctx, types.PP_NETWORK_KEY, bs.ppNetwork)
	bs.p2pServ.AddConnConntextKey(types.PP_NETWORK_KEY)

	bs.p2pServ.Start(ctx)
	_, _ = bs.p2pServ.ConnectToSP(ctx) // Ignore error if we can't connect to any SPs
	bs.ppNetwork.StartPP(ctx)
	return nil
}

func (bs *BaseServer) startTrafficLog() error {
	ctx := context.Background()
	ctx = context.WithValue(ctx, types.P2P_SERVER_KEY, bs.p2pServ)
	ctx = context.WithValue(ctx, types.PP_NETWORK_KEY, bs.ppNetwork)
	StartDumpTrafficLog(ctx)
	return nil
}

func (bs *BaseServer) startInternalApiServer() error {
	if setting.Config.WalletAddress != "" && setting.Config.InternalPort != "" {
		ctx := context.Background()
		ctx = context.WithValue(ctx, types.P2P_SERVER_KEY, bs.p2pServ)
		ctx = context.WithValue(ctx, types.PP_NETWORK_KEY, bs.ppNetwork)
		go api.StartHTTPServ(ctx)
	} else {
		utils.ErrorLog("Missing configuration for internal API server")
	}
	return nil
}

func (bs *BaseServer) startRestServer() error {
	if setting.Config.RestPort != "" {
		ctx := context.Background()
		ctx = context.WithValue(ctx, types.P2P_SERVER_KEY, bs.p2pServ)
		ctx = context.WithValue(ctx, types.PP_NETWORK_KEY, bs.ppNetwork)
		go rest.StartHTTPServ(ctx)
	} else {
		utils.ErrorLog("Missing configuration for rest port")
	}
	return nil
}

func (bs *BaseServer) Stop() {
	utils.DebugLogf("BaseServer.Stop ... ")
	if bs.ipcServ != nil {
		_ = bs.ipcServ.stop()
	}
	if bs.httpRpcServ != nil {
		bs.httpRpcServ.stop()
	}
	if bs.monitorServ != nil {
		bs.monitorServ.stop()
	}
	if bs.p2pServ != nil {
		bs.p2pServ.Stop()
	}
	// TODO: stop IPC, TrafficLog, InternalApiServer, RestServer
}
