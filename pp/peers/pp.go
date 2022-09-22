package peers

import (
	"context"
	"net"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/setting"
)

//todo: pp server should be move out of peers package
const (
	PP_LOG_ALL      = false
	PP_LOG_READ     = true
	PP_LOG_WRITE    = true
	PP_LOG_INBOUND  = true
	PP_LOG_OUTBOUND = true
)

// PPServer
type PPServer struct {
	*core.Server
	quitChs []chan bool
}

var ppServ *PPServer
var ppPeerClock = clock.NewClock()

// GetPPServer
func GetPPServer() *PPServer {
	return ppServ
}

// GetPPServer
func GetQuitChs() []chan bool {
	return ppServ.quitChs
}

func SetPPServer(pp *PPServer) {
	ppServ = pp
}

// StartListenServer
func StartListenServer(ctx context.Context, port string) {
	netListen, err := net.Listen(setting.PP_SERVER_TYPE, ":"+port)
	if err != nil {
		pp.ErrorLog(ctx, "StartListenServer", err)
	}
	spbServer := NewServer(ctx)
	ppServ = spbServer
	pp.Log(ctx, "StartListenServer!!! ", port)
	err = spbServer.Start(netListen)
	if err != nil {
		pp.ErrorLog(ctx, "StartListenServer Error", err)
	}
}

// NewServer returns a server.
func NewServer(ctx context.Context) *PPServer {
	onConnectOption := core.OnConnectOption(func(conn core.WriteCloser) bool {
		pp.Log(ctx, "on connect")
		return true
	})
	onErrorOption := core.OnErrorOption(func(conn core.WriteCloser) {
		pp.Log(ctx, "on error")
	})
	onCloseOption := core.OnCloseOption(func(conn core.WriteCloser) {
		netID := conn.(*core.ServerConn).GetNetID()
		peerList.PPDisconnectedNetId(ctx, netID)
	})

	maxConnection := setting.DEFAULT_MAX_CONNECTION
	if setting.Config.MaxConnection > maxConnection {
		maxConnection = setting.Config.MaxConnection
	}
	ppServer := &PPServer{core.CreateServer(
		onConnectOption,
		onErrorOption,
		onCloseOption,
		core.BufferSizeOption(10000),
		core.LogOpenOption(true),
		core.MinAppVersionOption(setting.Config.Version.MinAppVer),
		core.P2pAddressOption(setting.P2PAddress),
		core.MaxConnectionsOption(maxConnection)),
		make([]chan bool, 0),
	}

	ppServer.SetVolRecOptions(
		core.LogAllOption(PP_LOG_ALL),
		core.LogReadOption(PP_LOG_READ),
		core.OnWriteOption(PP_LOG_WRITE),
		core.LogInboundOption(PP_LOG_INBOUND),
		core.LogOutboundOption(PP_LOG_OUTBOUND),
		core.OnStartLogOption(func(s *core.Server) {
			pp.Log(ctx, "on start volume log")
			s.AddVolumeLogJob(PP_LOG_ALL, PP_LOG_READ, PP_LOG_WRITE, PP_LOG_INBOUND, PP_LOG_OUTBOUND)
		}),
	)

	return ppServer
}

func AppendQuitCh(qCh chan bool) {
	ppServ.quitChs = append(ppServ.quitChs, qCh)
}
