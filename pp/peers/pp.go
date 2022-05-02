package peers

import (
	"net"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
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
}

var ppServ *PPServer
var ppPeerClock = clock.NewClock()

// GetPPServer
func GetPPServer() *PPServer {
	return ppServ
}

func SetPPServer(pp *PPServer) {
	ppServ = pp
}

// StartListenServer
func StartListenServer(port string) {
	netListen, err := net.Listen("tcp4", ":"+port)
	if err != nil {
		utils.ErrorLog("StartListenServer", err)
	}
	spbServer := NewServer()
	ppServ = spbServer
	utils.Log("StartListenServer!!! ", port)
	err = spbServer.Start(netListen)
	if err != nil {
		utils.ErrorLog("StartListenServer Error", err)
	}
}

// NewServer returns a server.
func NewServer() *PPServer {
	onConnectOption := core.OnConnectOption(func(conn core.WriteCloser) bool {
		utils.Log("on connect")
		return true
	})
	onErrorOption := core.OnErrorOption(func(conn core.WriteCloser) {
		utils.Log("on error")
	})
	onCloseOption := core.OnCloseOption(func(conn core.WriteCloser) {
		netID := conn.(*core.ServerConn).GetNetID()
		peerList.PPDisconnectedNetId(netID)
	})

	ppServer := &PPServer{core.CreateServer(
		onConnectOption,
		onErrorOption,
		onCloseOption,
		core.BufferSizeOption(10000),
		core.LogOpenOption(true),
		core.MinAppVersionOption(setting.Config.Version.MinAppVer),
		core.P2pAddressOption(setting.P2PAddress)),
	}

	ppServer.SetVolRecOptions(
		core.LogAllOption(PP_LOG_ALL),
		core.LogReadOption(PP_LOG_READ),
		core.OnWriteOption(PP_LOG_WRITE),
		core.LogInboundOption(PP_LOG_INBOUND),
		core.LogOutboundOption(PP_LOG_OUTBOUND),
		core.OnStartLogOption(func(s *core.Server) {
			utils.Log("on start volume log")
			s.AddVolumeLogJob(PP_LOG_ALL, PP_LOG_READ, PP_LOG_WRITE, PP_LOG_INBOUND, PP_LOG_OUTBOUND)
		}),
	)

	return ppServer
}
