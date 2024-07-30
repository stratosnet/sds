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
	fwcryptotypes "github.com/stratosnet/sds/framework/crypto/types"
	"github.com/stratosnet/sds/framework/msg"
	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/sds-msg/protos"
	"google.golang.org/protobuf/proto"

	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
)

const (
	PP_LOG_ALL      = false
	PP_LOG_READ     = true
	PP_LOG_WRITE    = true
	PP_LOG_INBOUND  = true
	PP_LOG_OUTBOUND = true

	MIN_RECONNECT_INTERVAL_THRESHOLD = 60  // seconds
	MAX_RECONNECT_INTERVAL_THRESHOLD = 600 // seconds
	RECONNECT_INTERVAL_MULTIPLIER    = 2
)

type P2pServer struct {
	// server for pp to serve event messages
	server          *core.Server
	quitChMap       map[types.ContextKey]chan bool
	bufferedSpConns []*cf.ClientConn

	p2pPrivKey fwcryptotypes.PrivKey
	p2pPubKey  fwcryptotypes.PubKey
	p2pAddress fwtypes.P2PAddress

	// client conn
	// offlineChan
	offlineChan chan *offline

	// mainSpConn super node connection
	mainSpConn *cf.ClientConn

	// SPMaintenanceMap stores records of SpUnderMaintenance, K - SpP2pAddress, V - lastReconnectRecord
	SPMaintenanceMap *utils.AutoCleanMap

	// cachedConnMap upload connection
	cachedConnMap *sync.Map

	// connMap client connection map
	connMap map[string]*cf.ClientConn

	clientMutex sync.Mutex

	connContextKey []interface{}

	onWriteFunc  func(context.Context, *msg.RelayMsgBuf)
	onReadFunc   func(*msg.RelayMsgBuf)
	onHandleFunc func(context.Context, *msg.RelayMsgBuf)
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

	p2pKey, err := fwtypes.DecryptKey(p2pKeyFile, setting.Config.Keys.P2PPassword, false)
	if err != nil {
		return errors.New("couldn't decrypt P2P key file: " + err.Error())
	}

	p.p2pPrivKey = p2pKey.PrivateKey
	p.p2pPubKey = p.p2pPrivKey.PubKey()
	p.p2pAddress = fwtypes.P2PAddress(p.p2pPubKey.Address())
	p.SPMaintenanceMap = utils.NewAutoCleanMap(time.Duration(MAX_RECONNECT_INTERVAL_THRESHOLD) * time.Second)
	return nil
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
		pp.DebugLogf(ctx, "PP %v with netId %v is offline", conn.(*core.ServerConn).GetRemoteAddr(), conn.(*core.ServerConn).GetNetID())
	})
	onBadAppVerOption := core.OnBadAppVerOption(func(version uint16, cmd uint8, minAppVer uint16) []byte {
		return p.BuildBadVersionMsg(version, cmd, minAppVer)
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
		onBadAppVerOption,
		core.OnWriteOption(p.onWriteFunc),
		core.OnHandleOption(p.onHandleFunc),
		core.BufferSizeOption(10000),
		core.LogOpenOption(true),
		core.MinAppVersionOption(setting.Config.Version.MinAppVer),
		core.P2pAddressOption(p.GetP2PAddress().String()),
		core.MaxConnectionsOption(maxConnections),
		core.ContextKVOption(ckv),
	)
	server.SetVolRecOptions(
		core.LogAllOption(PP_LOG_ALL),
		core.LogReadOption(PP_LOG_READ),
		core.LogWriteOption(PP_LOG_WRITE),
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
	go p.StartListenServer(ctx, setting.GetP2pServerPort())
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

func (p *P2pServer) BuildBadVersionMsg(version uint16, cmd uint8, minAppVer uint16) []byte {
	req := &protos.RspBadVersion{
		Version:        int32(version),
		MinimumVersion: int32(minAppVer),
		Command:        uint32(cmd),
	}
	data, err := proto.Marshal(req)
	if err != nil {
		utils.ErrorLog(err)
		return nil
	}
	return data
}

func GetP2pServer(ctx context.Context) *P2pServer {
	if ctx == nil || ctx.Value(types.P2P_SERVER_KEY) == nil {
		panic("P2pServer is not instantiated")
	}
	ps := ctx.Value(types.P2P_SERVER_KEY).(*P2pServer)
	return ps
}
