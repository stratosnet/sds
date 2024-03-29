package p2pserver

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
	utilstypes "github.com/stratosnet/sds/utils/types"
)

const (
	PP_LOG_ALL      = false
	PP_LOG_READ     = true
	PP_LOG_WRITE    = true
	PP_LOG_INBOUND  = true
	PP_LOG_OUTBOUND = true

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
	quitChMap       map[types.ContextKey]chan bool
	peerList        types.PeerList
	bufferedSpConns []*cf.ClientConn

	p2pPrivKey utilstypes.P2pPrivKey
	p2pPubKey  utilstypes.P2pPubKey
	p2pAddress utilstypes.Address

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

func (p *P2pServer) GetP2pServer() *core.Server {
	return p.server
}

func (p *P2pServer) SetPPServer(pp *core.Server) {
	p.server = pp
}

func (p *P2pServer) Init() error {
	p2pKeyFile, err := os.ReadFile(filepath.Join(setting.Config.Home.AccountsPath, setting.Config.Keys.P2PAddress+".json"))
	if err != nil {
		return errors.New("couldn't read P2P key file: " + err.Error())
	}

	p2pKey, err := utils.DecryptKey(p2pKeyFile, setting.Config.Keys.P2PPassword)
	if err != nil {
		return errors.New("couldn't decrypt P2P key file: " + err.Error())
	}

	p.p2pPrivKey = utilstypes.BytesToP2pPrivKey(p2pKey.PrivateKey)
	p.p2pPubKey = p.p2pPrivKey.PubKey()
	p.p2pAddress, err = utilstypes.P2pAddressFromBech(setting.Config.Keys.P2PAddress)
	return err
}

func (p *P2pServer) StartListenServer(ctx context.Context, port string) {
	netListen, err := net.Listen(setting.P2pServerType, ":"+port)
	if err != nil {
		utils.ErrorLog("StartListenServer Error", err)
		return
	}
	spbServer := p.newServer(ctx)
	p.server = spbServer
	utils.DebugLog("StartListenServer!!! ", port)
	err = spbServer.Start(netListen)
	if err != nil {
		utils.ErrorLog("StartListenServer Error", err)
	}
}

// newServer returns a server.
func (p *P2pServer) newServer(ctx context.Context) *core.Server {
	onConnectOption := core.OnConnectOption(func(conn core.WriteCloser) bool { return true })
	onErrorOption := core.OnErrorOption(func(conn core.WriteCloser) {})
	onCloseOption := core.OnCloseOption(func(conn core.WriteCloser) {
		netID := conn.(*core.ServerConn).GetNetID()
		p.PPDisconnectedNetId(ctx, netID)
	})

	maxConnections := setting.DefaultMaxConnections
	if setting.Config.Traffic.MaxConnections > maxConnections {
		maxConnections = setting.Config.Traffic.MaxConnections
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
		core.P2pAddressOption(p.GetP2PAddress()),
		core.MaxConnectionsOption(maxConnections),
		core.ContextKVOption(ckv),
	)
	server.SetVolRecOptions(
		core.LogAllOption(PP_LOG_ALL),
		core.LogReadOption(PP_LOG_READ),
		core.OnWriteOption(PP_LOG_WRITE),
		core.LogInboundOption(PP_LOG_INBOUND),
		core.LogOutboundOption(PP_LOG_OUTBOUND),
		core.OnStartLogOption(func(s *core.Server) {
			s.AddVolumeLogJob(PP_LOG_ALL, PP_LOG_READ, PP_LOG_WRITE, PP_LOG_INBOUND, PP_LOG_OUTBOUND)
		}),
	)

	return server
}

func (p *P2pServer) Start(ctx context.Context) {
	// channels for quitting peer level goroutines
	utils.InitBufferPool(setting.MaxData, setting.GetDataBufferSize())

	ctx = p.initQuitChs(ctx)
	setting.SetMyNetworkAddress()
	p.peerList.Init(setting.NetworkAddress, filepath.Join(setting.Config.Home.PeersPath, "pp-list"))
	go p.StartListenServer(ctx, setting.Config.Node.Connectivity.NetworkPort)
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

func (p *P2pServer) initQuitChs(ctx context.Context) context.Context {
	p.quitChMap = make(map[types.ContextKey]chan bool)
	quitChListenOffline := make(chan bool, 1)
	ctx = context.WithValue(ctx, types.LISTEN_OFFLINE_QUIT_CH_KEY, quitChListenOffline)
	p.quitChMap[types.LISTEN_OFFLINE_QUIT_CH_KEY] = quitChListenOffline
	return ctx
}

func (p *P2pServer) AddConnConntextKey(key interface{}) {
	p.connContextKey = append(p.connContextKey, key)
}

func GetP2pServer(ctx context.Context) *P2pServer {
	if ctx == nil || ctx.Value(types.P2P_SERVER_KEY) == nil {
		panic("P2pServer is not instantiated")
	}
	ps := ctx.Value(types.P2P_SERVER_KEY).(*P2pServer)
	return ps
}
