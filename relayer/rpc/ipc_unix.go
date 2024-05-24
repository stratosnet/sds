//go:build darwin || dragonfly || freebsd || linux || nacl || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package rpc

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/stratosnet/sds/framework/utils"
)

// ipcListen will create a Unix socket on the given endpoint.
func ipcListen(endpoint string) (net.Listener, error) {
	if len(endpoint) > int(max_path_size) {
		utils.WarnLog(fmt.Sprintf("The ipc endpoint is longer than %d characters. ", max_path_size),
			"endpoint", endpoint)
	}

	// Ensure the IPC path exists and remove any previous leftover
	if err := os.MkdirAll(filepath.Dir(endpoint), 0751); err != nil {
		return nil, err
	}
	_ = os.Remove(endpoint)
	l, err := net.Listen("unix", endpoint)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(endpoint, 0600); err != nil {
		return nil, err
	}
	return l, nil
}

// newIPCConnection will connect to a Unix socket on the given endpoint.
func newIPCConnection(ctx context.Context, endpoint string) (net.Conn, error) {
	return new(net.Dialer).DialContext(ctx, "unix", endpoint)
}
