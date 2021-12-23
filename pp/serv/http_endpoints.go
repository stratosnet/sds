package serv

import (
	"net"
	"net/http"
	"time"

	"github.com/stratosnet/sds/rpc"
	"github.com/stratosnet/sds/utils"
)

// StartHTTPEndpoint starts the HTTP RPC endpoint.
func StartHTTPEndpoint(endpoint string, timeouts rpc.HTTPTimeouts, handler http.Handler) (*http.Server, net.Addr, error) {
	// start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return nil, nil, err
	}
	// make sure timeout values are meaningful
	CheckTimeouts(&timeouts)
	// Bundle and start the HTTP server
	httpSrv := &http.Server{
		Handler:      handler,
		ReadTimeout:  timeouts.ReadTimeout,
		WriteTimeout: timeouts.WriteTimeout,
		IdleTimeout:  timeouts.IdleTimeout,
	}
	go httpSrv.Serve(listener)
	return httpSrv, listener.Addr(), err
}

// checkModuleAvailability checks that all names given in modules are actually
// available API services. It assumes that the MetadataApi module ("rpc") is always available;
// the registration of this "rpc" module happens in NewServer() and is thus common to all endpoints.
func checkModuleAvailability(modules []string, apis []rpc.API) (bad, available []string) {
	availableSet := make(map[string]struct{})
	for _, api := range apis {
		if _, ok := availableSet[api.Namespace]; !ok {
			availableSet[api.Namespace] = struct{}{}
			available = append(available, api.Namespace)
		}
	}
	for _, name := range modules {
		if _, ok := availableSet[name]; !ok && name != rpc.MetadataApi {
			bad = append(bad, name)
		}
	}
	return bad, available
}

// CheckTimeouts ensures that timeout values are meaningful
func CheckTimeouts(timeouts *rpc.HTTPTimeouts) {
	if timeouts.ReadTimeout < time.Second {
		utils.WarnLog("Sanitizing invalid HTTP read timeout", "provided", timeouts.ReadTimeout, "updated", rpc.DefaultHTTPTimeouts.ReadTimeout)
		timeouts.ReadTimeout = rpc.DefaultHTTPTimeouts.ReadTimeout
	}
	if timeouts.WriteTimeout < time.Second {
		utils.WarnLog("Sanitizing invalid HTTP write timeout", "provided", timeouts.WriteTimeout, "updated", rpc.DefaultHTTPTimeouts.WriteTimeout)
		timeouts.WriteTimeout = rpc.DefaultHTTPTimeouts.WriteTimeout
	}
	if timeouts.IdleTimeout < time.Second {
		utils.WarnLog("Sanitizing invalid HTTP idle timeout", "provided", timeouts.IdleTimeout, "updated", rpc.DefaultHTTPTimeouts.IdleTimeout)
		timeouts.IdleTimeout = rpc.DefaultHTTPTimeouts.IdleTimeout
	}
}
