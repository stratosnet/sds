package net

import (
	"context"
	"errors"
	"fmt"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/relay/stratoschain"
	"github.com/stratosnet/sds/relay/stratoschain/prefix"
	"github.com/stratosnet/sds/sp/common"
	"github.com/stratosnet/sds/sp/storages"
	"github.com/stratosnet/sds/sp/storages/data"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/cache"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	"github.com/stratosnet/sds/utils/database"
	"github.com/stratosnet/sds/utils/hashring"
	"github.com/tendermint/tendermint/crypto"
	"io/ioutil"
	"net"
	"sync"
	"time"

	"github.com/alex023/clock"

	"github.com/golang/protobuf/proto"
)

// Server sp server
type Server struct {
	Ver                uint16               // version
	PPVersion          uint16               // PP version
	Host               string               // net host
	WalletPrivateKey   crypto.PrivKey       // Wallet private key
	P2PPrivateKey      []byte               // P2P private key
	UserCount          int64                // user count todo should this be atomic?
	ConnectedCount     uint64               // connection count todo should this be atomic?
	Conf               *Config              // configuration
	CT                 *database.CacheTable // database
	HashRing           *hashring.HashRing   // hashring
	serv               *spbf.Server         // server
	connPool           *sync.Map            // connection pool
	msgHandler         *MsgHandler
	System             *data.System
	SubscriptionServer *SubscriptionServer // Server for websocket subscriptions

	storages.ServerCache
	sync.Mutex
}

// initialize
func (s *Server) initialize() error {

	if s.Conf == nil {
		return errors.New("wrong config")
	}

	logger := utils.NewDefaultLogger(s.Conf.Log.Path, s.Conf.Log.OutputStd, s.Conf.Log.OutputFile)
	logger.SetLogLevel(utils.LogLevel(s.Conf.Log.Level))

	if s.Conf.Net.Host == "" {
		return errors.New("missing host config")
	}

	utils.Log("initializing...")
	err := s.verifyNodeKeys()
	if err != nil {
		utils.ErrorLog("couldn't load P2P and wallet keys: ", err)
		return err
	}

	s.Host = s.Conf.Net.Host + ":" + s.Conf.Net.Port

	utils.Log("listen: ", s.Host)

	s.Ver = s.Conf.Version
	s.HashRing = hashring.New(s.Conf.HashRing.VirtualNodeNum)
	s.Cache = cache.NewRedis(s.Conf.Cache)
	if s.Conf.Peers.List == 0 {
		s.Conf.Peers.List = 10
	}
	if s.CT == nil {
		s.CT = database.NewCacheTable(s.Cache, s.Conf.Database)
		s.CT.GetDriver().SetCacheEngine(s.Cache)
	}

	s.CT.GetDriver().GetDB().Exec("SET NAMES utf8mb4")

	s.System = &data.System{}
	s.Load(s.System)
	s.System.InviteReward = s.Conf.User.InviteReward
	s.System.UpgradeReward = s.Conf.User.UpgradeReward
	s.System.InitializeCapacity = s.Conf.User.InitializeCapacity
	s.Store(s.System, 0)

	s.msgHandler = NewMsgHandler(s)

	go s.msgHandler.Run()

	// it's commented out
	//s.puk = tools.LoadOrCreateAccount(s.Conf.Keys.WalletPath, s.Conf.Keys.WalletPassword)

	s.connPool = new(sync.Map)

	s.UserCount, _ = s.CT.CountTable(new(table.PP), map[string]interface{}{})

	// register heartbeat response
	spbf.Register(header.ReqHeart, func(ctx context.Context, conn spbf.WriteCloser) {
		m := msg.RelayMsgBuf{
			MSGHead: header.MakeMessageHeader(1, s.Ver, 0, header.RspHeart),
		}
		conn.Write(&m)
	})

	s.BuildHashRing()
	return nil
}

// BuildHashRing
func (s *Server) BuildHashRing() {

	// clean online PP status
	updateData := map[string]interface{}{"state": table.STATE_OFFLINE}
	updateWhere := map[string]interface{}{"state = ?": table.STATE_ONLINE}
	s.CT.GetDriver().Update("pp", updateData, updateWhere)

	// initialize hashring
	ppList, err := s.CT.GetDriver().FetchAll("pp", map[string]interface{}{
		"columns": "p2p_address, network_address, pub_key",
	})
	if err == nil {
		for _, pp := range ppList {
			node := &hashring.Node{
				ID:   pp["p2p_address"].(string),
				Host: pp["network_address"].(string),
			}
			s.HashRing.AddNode(node)
		}
	}
}

// ListenEvent
func (s *Server) ListenEvent(cmd string, event Event) {
	event.SetServer(s)
	spbf.Register(cmd, event.Handle)
}

// HandleMsg
func (s *Server) HandleMsg(message common.Msg) {
	go s.msgHandler.AddMsg(message)
}

//refreshStatus refresh sp status
func (s *Server) refreshStatus() {
	c := clock.NewClock()

	logger := utils.NewLogger("tmp/logs/mirror.log", false, true)
	logger.SetLogLevel(utils.Debug)

	// update sp status: every 3 second
	c.AddJobRepeat(time.Second*3, 0, func() {
		if s.Load(s.System) == nil {
			s.System.Connected = s.ConnectedCount
			s.System.User = uint64(s.UserCount)
			s.System.OnlinePPCount = uint64(s.HashRing.NodeCount)
			s.PPVersion = s.System.Version
			s.Store(s.System, 0)
		}

		log := fmt.Sprintln()
		log = log + fmt.Sprintln("+----------------------------------------------------------------------")
		log = log + fmt.Sprintln("| SP ver:", s.Ver, "| PP ver:", s.PPVersion)
		log = log + fmt.Sprintln("+----------------------------------------------------------------------")
		log = log + fmt.Sprintln("| connection count：", s.ConnectedCount)
		log = log + fmt.Sprintln("| user count：", s.UserCount)
		log = log + fmt.Sprintln("| online PP count：", s.HashRing.NodeOkCount)
		log = log + fmt.Sprintln("+----------------------------------------------------------------------")

		logger.Log(utils.Info, log)
	})

	// refresh all useful info, every 10 minutes
	c.AddJobRepeat(time.Second*10*60, 0, func() {

		// eg: file download leader board
	})
}

// AddConn
func (s *Server) AddConn(name, p2pAddress string, conn spbf.WriteCloser) {
	s.connPool.Store(name, p2pAddress)
	s.connPool.Store(p2pAddress+"#name", name)
	s.connPool.Store(p2pAddress+"#connect", conn)
	s.ConnectedCount++
}

// RmConn
func (s *Server) RmConn(name string) {
	p2pAddress := s.Who(name)
	s.connPool.Delete(name)
	s.connPool.Delete(p2pAddress + "#name")
	s.connPool.Delete(p2pAddress + "#connect")
	s.ConnectedCount--
}

// GetConn
func (s *Server) GetConn(p2pAddress string) spbf.WriteCloser {
	if c, ok := s.connPool.Load(p2pAddress + "#connect"); ok {
		return c.(spbf.WriteCloser)
	}
	return nil
}

// GetName
func (s *Server) GetName(p2pAddress string) string {
	if n, ok := s.connPool.Load(p2pAddress + "#name"); ok {
		return n.(string)
	}
	return ""
}

// Who returns the P2P key address, can be used to check if a node is a PP.
func (s *Server) Who(name string) string {
	if wa, ok := s.connPool.Load(name); ok {
		return wa.(string)
	}
	return ""
}

// SendMsg send msg to PP
func (s *Server) SendMsg(p2pAddress string, cmd string, message proto.Message) {
	conn := s.GetConn(p2pAddress)
	if conn == nil {
		return
	}
	d, err := proto.Marshal(message)
	if err != nil {
		utils.Log(err)
	}
	msg := &msg.RelayMsgBuf{
		MSGHead: header.MakeMessageHeader(1, s.Ver, uint32(len(d)), cmd),
		MSGData: d,
	}
	conn.Write(msg)

}

// Start as SP
func (s *Server) Start() {

	// initialization
	err := s.initialize()
	if err != nil {
		utils.ErrorLogf("Initializing SP server error : %v", err)
		return
	}

	// refresh status
	go s.refreshStatus()

	// Starts subscriptions websocket server
	s.SubscriptionServer = NewSubscriptionServer(s)
	s.SubscriptionServer.Start()
	defer s.SubscriptionServer.Close()

	// start listening
	netListen, err := net.Listen("tcp", s.Host)
	if err != nil {
		utils.ErrorLogf("error creating SP server tcp connection: %v", err)
	}

	_ = s.serv.Start(netListen)
}

// OnConnectOption
func (s *Server) OnConnectOption(_ spbf.WriteCloser) bool {
	return true
}

// OnCloseOption
func (s *Server) OnCloseOption(conn spbf.WriteCloser) {

	go s.HandleMsg(&common.MsgLogout{
		Name: conn.(*spbf.ServerConn).GetName(),
	})
}

// NewServer
func NewServer(configFilePath string) *Server {

	if configFilePath == "" {
		utils.ErrorLog("missing config file")
		return nil
	}

	server := &Server{Conf: &Config{}}

	utils.LoadYamlConfig(server.Conf, configFilePath)
	if server.Conf == nil {
		utils.ErrorLog("wrong config given")
		return nil
	}

	server.serv = spbf.CreateServer(
		spbf.OnConnectOption(server.OnConnectOption),
		spbf.OnCloseOption(server.OnCloseOption),
		spbf.MaxConnectionsOption(2000000),
		spbf.MaxFlowOption(125*1024*1024),
	)

	prefix.SetConfig(server.Conf.BlockchainInfo.AddressPrefix)
	stratoschain.Url = "http://" + server.Conf.BlockchainInfo.StratosChainAddress + ":" + server.Conf.BlockchainInfo.StratosChainPort

	return server
}

func (s *Server) NewMsgHandler() {
	s.msgHandler = NewMsgHandler(s)
	go s.msgHandler.Run()
}

func (s *Server) NewConnPool() {
	s.connPool = &sync.Map{}
}

func (s *Server) CreateServ() {
	s.serv = spbf.CreateServer()
}

func (s *Server) StartServ(listener net.Listener) error {
	return s.serv.Start(listener)
}

func (s *Server) verifyNodeKeys() error {
	p2pJson, err := ioutil.ReadFile(s.Conf.Keys.P2PPath)
	if err != nil {
		return err
	}

	p2pKey, err := utils.DecryptKey(p2pJson, s.Conf.Keys.P2PPassword)
	if err != nil {
		return err
	}

	s.P2PPrivateKey = p2pKey.PrivateKey
	utils.DebugLog("verified P2P key successfully!")

	walletJson, err := ioutil.ReadFile(s.Conf.Keys.WalletPath)
	if err != nil {
		return err
	}

	walletKey, err := utils.DecryptKey(walletJson, s.Conf.Keys.WalletPassword)
	if err != nil {
		return err
	}

	s.WalletPrivateKey = secp256k1.PrivKeyBytesToTendermint(walletKey.PrivateKey)
	utils.DebugLog("verified wallet key successfully!")
	return nil
}
