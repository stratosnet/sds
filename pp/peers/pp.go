package peers

import (
	"context"
	"net"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/pp"
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

//type WritePacketCostTime struct {
//	reqId 	 string
//	costTime int64
//}

var (
//CostTimeCh = make(chan *core.WritePacketCostTime)
// Maps to record uploading stats
//ReqIdMap = &sync.Map{} // K: reqId, V: {tkId+sliceNum, up/down}
//UpSendCostTimeMap = &sync.Map{} // K: tkId+sliceNum, V: int64 (timecost)
//UpSendPacketWgMap = &sync.Map{} // K: tkId+sliceNum, V: waitGroup
//
//UpRecvCostTimeMap = &sync.Map{} // K: tkId+sliceNum, V: int64 (timecost)
//UpRecvPacketWgMap = &sync.Map{} // K: tkId+sliceNum, V: waitGroup
//
//DownSendCostTimeMap = &sync.Map{} // K: tkId+sliceNum, V: int64 (timecost)
//DownSendPacketWgMap = &sync.Map{} // K: tkId+sliceNum, V: waitGroup
//
//DownRecvCostTimeMap = &sync.Map{} // K: tkId+sliceNum, V: int64 (timecost)
//DownRecvPacketWgMap = &sync.Map{} // K: tkId+sliceNum, V: waitGroup
)

func ListenSendPacket(handler func(core.WritePacketCostTime)) {
	for {
		select {
		case entry, ok := <-core.CostTimeCh:
			if ok && entry.CostTime > 0 && entry.ReqId > 0 {
				utils.DebugLogf("received report from WritePacket: %v", entry)
				handler(entry)
			}
		}
	}
}
