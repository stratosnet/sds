package net

import (
	"context"
	"fmt"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/sp/common"
	"github.com/stratosnet/sds/sp/storages"
	"github.com/stratosnet/sds/sp/storages/data"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/cache"
	"github.com/stratosnet/sds/utils/database"
	"github.com/stratosnet/sds/utils/hashring"
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
	puk                string               // public key
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
func (s *Server) initialize() {

	if s.Conf == nil {
		utils.ErrorLog("wrong config")
		return
	}

	if s.Conf.Net.Host == "" {
		utils.ErrorLog("missing host, start fail")
		return
	}

	utils.Log("initializing...")

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

	s.System = new(data.System)
	s.Load(s.System)
	s.System.InviteReward = s.Conf.User.InviteReward
	s.System.UpgradeReward = s.Conf.User.UpgradeReward
	s.System.InitializeCapacity = s.Conf.User.InitializeCapacity
	s.Store(s.System, 0)

	s.msgHandler = NewMsgHandler(s)

	go s.msgHandler.Run()

	// it's commented out
	//s.puk = tools.LoadOrCreateAccount(s.Conf.Ecdsa.PrivateKeyPath, s.Conf.Ecdsa.PrivateKeyPass)

	s.connPool = new(sync.Map)

	s.UserCount, _ = s.CT.CountTable(new(table.PP), map[string]interface{}{})

	// register heartbeat response
	spbf.Register(header.ReqHeart, func(ctx context.Context, conn spbf.WriteCloser) {
		msg := msg.RelayMsgBuf{
			MSGHead: header.MakeMessageHeader(1, s.Ver, 0, header.RspHeart),
		}
		conn.Write(&msg)
	})

	s.BuildHashRing()
}

// BuildHashRing
func (s *Server) BuildHashRing() {

	// clean online PP status
	updateData := map[string]interface{}{"state": table.STATE_OFFLINE}
	updateWhere := map[string]interface{}{"state = ?": table.STATE_ONLINE}
	s.CT.GetDriver().Update("pp", updateData, updateWhere)

	// initialize hashring
	ppList, err := s.CT.GetDriver().FetchAll("pp", map[string]interface{}{
		"columns": "wallet_address, network_address, pub_key",
	})
	if err == nil {
		for _, pp := range ppList {
			node := &hashring.Node{
				ID:   pp["wallet_address"].(string),
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
func (s *Server) AddConn(name, walletAddress string, conn spbf.WriteCloser) {
	s.connPool.Store(name, walletAddress)
	s.connPool.Store(walletAddress+"#name", name)
	s.connPool.Store(walletAddress+"#connect", conn)
	s.ConnectedCount++
}

// RmConn
func (s *Server) RmConn(name string) {
	walletAddress := s.Who(name)
	s.connPool.Delete(name)
	s.connPool.Delete(walletAddress + "#name")
	s.connPool.Delete(walletAddress + "#connect")
	s.ConnectedCount--
}

// GetConn
func (s *Server) GetConn(walletAddress string) spbf.WriteCloser {
	if c, ok := s.connPool.Load(walletAddress + "#connect"); ok {
		return c.(spbf.WriteCloser)
	}
	return nil
}

// GetName
func (s *Server) GetName(walletAddress string) string {
	if n, ok := s.connPool.Load(walletAddress + "#name"); ok {
		return n.(string)
	}
	return ""
}

// Who return wallet address, can be used to check if PP.
func (s *Server) Who(name string) string {
	if wa, ok := s.connPool.Load(name); ok {
		return wa.(string)
	}
	return ""
}

// SendMsg send msg to PP
func (s *Server) SendMsg(walletAddress string, cmd string, message proto.Message) {
	conn := s.GetConn(walletAddress)
	if conn != nil {
		data, err := proto.Marshal(message)
		if utils.CheckError(err) {
			utils.Log(err)
		}
		msg := &msg.RelayMsgBuf{
			MSGHead: header.MakeMessageHeader(1, s.Ver, uint32(len(data)), cmd),
			MSGData: data,
		}
		conn.Write(msg)
	}
}

// Start as SP
func (s *Server) Start() {

	// initialization
	s.initialize()

	// refresh status
	go s.refreshStatus()

	// Starts subscriptions websocket server
	s.SubscriptionServer = NewSubscriptionServer(s)
	s.SubscriptionServer.Start()
	defer s.SubscriptionServer.Close()

	// start listening
	netListen, err := net.Listen("tcp", s.Host)
	utils.CheckError(err)

	s.serv.Start(netListen)
}

// OnConnectOption
func (s *Server) OnConnectOption(conn spbf.WriteCloser) bool {
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

	server := new(Server)

	server.Conf = new(Config)
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

	return server
}
