package rpc

import (
	"context"
	"net"
	"strings"

	"github.com/stratosnet/sds/relay/utils"
)

// StartIPCEndpoint starts an IPC endpoint.
func StartIPCEndpoint(ipcEndpoint string, apis []API, ctx context.Context) (net.Listener, *Server, error) {
	// Register all the APIs exposed by the services.
	var (
		handler    = NewServer()
		regMap     = make(map[string]struct{})
		registered []string
	)
	for _, api := range apis {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			utils.Log("IPC registration failed", "namespace", api.Namespace, "error", err)
			return nil, nil, err
		}
		if _, ok := regMap[api.Namespace]; !ok {
			registered = append(registered, api.Namespace)
			regMap[api.Namespace] = struct{}{}
		}
	}
	utils.DebugLog("IPCs registered", "namespaces", strings.Join(registered, ","))
	// All APIs registered, start the IPC listener.
	listener, err := ipcListen(ipcEndpoint)
	if err != nil {
		return nil, nil, err
	}
	go func() {
		err := handler.ServeListener(listener, ctx)
		if err != nil {
			utils.ErrorLog("Error serving IPC listener", err)
		}
	}()
	return listener, handler, nil
}
