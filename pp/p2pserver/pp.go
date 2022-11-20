package p2pserver

import (
	"context"
	"net"
	"path/filepath"
	"sync"
	"time"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
)

const (
	PP_LOG_ALL      = false
	PP_LOG_READ     = true
	PP_LOG_WRITE    = true
	PP_LOG_INBOUND  = true
	PP_LOG_OUTBOUND = true

	P2P_SERVER_KEY             = "PPServerKey"
	LISTEN_OFFLINE_QUIT_CH_KEY = "ListenOfflineQuitCh"

	LAST_RECONNECT_KEY               = "last_reconnect"
	MIN_RECONNECT_INTERVAL_THRESHOLD = 60  // seconds
	MAX_RECONNECT_INTERVAL_THRESHOLD = 600 // seconds
	RECONNECT_INTERVAL_MULTIPLIER    = 2
)

type LastReconnectRecord struct {
	SpP2PAddress                string
	Time                        time.Time
	NextAllowableReconnectInSec int64
}
type P2pServer struct {
	// server for pp to serve event messages
	server          *core.Server
	quitChMap       map[string]chan bool
	peerList        types.PeerList
	bufferedSpConns []*cf.ClientConn

	// client conn
	// offlineChan
	offlineChan chan *offline

	// mainSpConn super node connection
	mainSpConn *cf.ClientConn

	// SPMaintenanceMap stores records of SpUnderMaintenance, K - SpP2pAddress, V - list of MaintenanceRecord
	SPMaintenanceMap *utils.AutoCleanMap

	// ppConn current connected pp node
	ppConn *cf.ClientConn

	// cachedConnMap upload connection
	cachedConnMap *sync.Map

	// connMap client connection map
	connMap map[string]*cf.ClientConn

	clientMutex sync.Mutex

	connContextKey []interface{}
}

// GetP2pServer
func (p *P2pServer) GetP2pServer() *core.Server {
	return p.server
}

// SetPPServer
func (p *P2pServer) SetPPServer(pp *core.Server) {
	p.server = pp
}

func (p *P2pServer) GetMainSpConn() *cf.ClientConn {
	return p.mainSpConn
}

// StartListenServer
func (p *P2pServer) StartListenServer(ctx context.Context, port string) {
	netListen, err := net.Listen(setting.PP_SERVER_TYPE, ":"+port)
	if err != nil {
		pp.ErrorLog(ctx, "StartListenServer", err)
	}
	spbServer := p.newServer(ctx)
	p.server = spbServer
	pp.Log(ctx, "StartListenServer!!! ", port)
	err = spbServer.Start(netListen)
	if err != nil {
		pp.ErrorLog(ctx, "StartListenServer Error", err)
	}
}

// newServer returns a server.
func (p *P2pServer) newServer(ctx context.Context) *core.Server {
	onConnectOption := core.OnConnectOption(func(conn core.WriteCloser) bool {
		pp.Log(ctx, "on connect")
		return true
	})
	onErrorOption := core.OnErrorOption(func(conn core.WriteCloser) {
		pp.Log(ctx, "on error")
	})
	onCloseOption := core.OnCloseOption(func(conn core.WriteCloser) {
		netID := conn.(*core.ServerConn).GetNetID()
		p.PPDisconnectedNetId(ctx, netID)
	})

	maxConnection := setting.DEFAULT_MAX_CONNECTION
	if setting.Config.MaxConnection > maxConnection {
		maxConnection = setting.Config.MaxConnection
	}
	var ckv []core.ContextKV
	for _, key := range p.connContextKey {
		ckv = append(ckv, core.ContextKV{Key: key, Value: ctx.Value(key)})
	}
	server := core.CreateServer(onConnectOption,
		onErrorOption,
		onCloseOption,
		core.BufferSizeOption(10000),
		core.LogOpenOption(true),
		core.MinAppVersionOption(setting.Config.Version.MinAppVer),
		core.P2pAddressOption(setting.P2PAddress),
		core.MaxConnectionsOption(maxConnection),
		core.ContextKVOption(ckv),
	)
	server.SetVolRecOptions(
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

	return server
}

func (p *P2pServer) Start(ctx context.Context) {
	// channels for quitting peer level goroutines
	ctx = p.initQuitChs(ctx)
	setting.SetMyNetworkAddress()
	p.peerList.Init(setting.NetworkAddress, filepath.Join(setting.Config.PPListDir, "pp-list"))
	go p.StartListenServer(ctx, setting.Config.Port)
	p.initClient()
}

func (p *P2pServer) Stop() {
	if p.server != nil {
		// send signal to close network level goroutines
		for _, ch := range p.quitChMap {
			ch <- true
		}
		p.server.Stop()
	}
}

// initQuitChs
func (p *P2pServer) initQuitChs(ctx context.Context) context.Context {
	p.quitChMap = make(map[string]chan bool)
	quitChListenOffline := make(chan bool, 1)
	ctx = context.WithValue(ctx, LISTEN_OFFLINE_QUIT_CH_KEY, quitChListenOffline)
	p.quitChMap[LISTEN_OFFLINE_QUIT_CH_KEY] = quitChListenOffline
	return ctx
}

func (p *P2pServer) AddConnConntextKey(key interface{}) {
	p.connContextKey = append(p.connContextKey, key)
}

// GetP2pServer
func GetP2pServer(ctx context.Context) *P2pServer {
	if ctx == nil || ctx.Value(P2P_SERVER_KEY) == nil {
		panic("P2pServer is not instantiated")
	}
	ps := ctx.Value(P2P_SERVER_KEY).(*P2pServer)
	return ps
}
