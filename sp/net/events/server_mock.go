package events

import (
	"context"
	"io/ioutil"
	"log"
	gonet "net"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/data"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/cache"
	"github.com/stratosnet/sds/utils/database"
	"github.com/stratosnet/sds/utils/database/config"
	"github.com/stratosnet/sds/utils/hashring"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// initialize
func initializeMock(s *net.Server) {

	if s.Conf == nil {
		utils.ErrorLog("wrong config")
		return
	}

	if s.Conf.Net.Host == "" {
		utils.ErrorLog("missing host, start fail")
		return
	}

	s.Host = s.Conf.Net.Host + ":" + s.Conf.Net.Port
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

	_, _ = s.CT.GetDriver().GetDB().Exec("SET NAMES utf8mb4")

	s.System = &data.System{}
	_ = s.Load(s.System)
	s.System.InviteReward = s.Conf.User.InviteReward
	s.System.UpgradeReward = s.Conf.User.UpgradeReward
	s.System.InitializeCapacity = s.Conf.User.InitializeCapacity
	_ = s.Store(s.System, 0)

	s.NewMsgHandler()

	// it's commented out
	//s.puk = tools.LoadOrCreateAccount(s.Conf.Ecdsa.PrivateKeyPath, s.Conf.Ecdsa.PrivateKeyPass)

	s.NewConnPool()

	s.UserCount, _ = s.CT.CountTable(new(table.PP), map[string]interface{}{})

	// register heartbeat response
	spbf.Register(header.ReqHeart, func(ctx context.Context, conn spbf.WriteCloser) {
		m := msg.RelayMsgBuf{
			MSGHead: header.MakeMessageHeader(1, s.Ver, 0, header.RspHeart),
		}
		_ = conn.Write(&m)
	})

	s.BuildHashRing()
}

// StartMock Start a mock server for test
func StartMock(cmd string, eventHandleFunc func(s *net.Server) EventHandleFunc) (s *net.Server, dbCloseFunc func(), redisCloseFunc func()) {
	tmpDir, err := ioutil.TempDir("", "logs")
	if err != nil {
		log.Panic(err)
	}

	utils.NewLogger(tmpDir+"/stdout.log", true, true)

	mysqlC := config.Connect{
		Driver:  "mysql",
		User:    "root",
		Pass:    "111111",
		DbName:  "sds",
		Debug:   false,
		LogFile: tmpDir + "/database.log",
	}

	mysqlC.Host, mysqlC.Port, dbCloseFunc = createTestMySQLContainer(mysqlC.DbName, mysqlC.User, mysqlC.Pass)

	redisConfig := cache.Config{
		Engine:   "redis",
		Host:     "", // set by container
		Port:     "", // set by container
		Pass:     "",
		DB:       0,
		LifeTime: 60,
	}
	redisCloseFunc = createTestRedisContainer(&redisConfig)
	// initialization
	conf := &net.Config{
		Version: 0,
		Net: net.NetworkConfig{
			Host: "127.0.0.1",
			Port: "10086",
		},
		Peers: net.PeersConfig{
			List:             10,
			RegisterSwitch:   false,
			ProvideDiskScale: 0,
		},
		HashRing: net.HashRingConfig{
			VirtualNodeNum: 2,
		},
		FileStorage: net.FileStorageConfig{},
		Cache:       redisConfig,
		Database:    mysqlC,
		Ecdsa: net.EcdsaConfig{
			PrivateKeyPath: "",
			PrivateKeyPass: "",
		},
		User: net.UserConfig{
			UpgradeReward:      0,
			InviteReward:       0,
			InitializeCapacity: 0,
		},
	}
	s = &net.Server{
		Conf: conf,
	}

	s.CreateServ()

	spbf.Register(cmd, eventHandleFunc(s))

	initializeMock(s)

	// refresh status
	//go s.refreshStatus()

	// start listening
	go func() {
		netListen, err := gonet.Listen("tcp", s.Host)
		if err != nil {
			log.Fatal(err)
			return
		}

		if err = s.StartServ(netListen); err != nil {
			log.Fatal(err)
		}
	}()

	return
}

func createTestMySQLContainer(db string, user string, pw string) (host string, port uint16, closeFunc func()) {
	log.Println("setup MySQL Container")
	ctx := context.Background()

	seedDataPath, err := os.Getwd()
	seedDataPath = strings.ReplaceAll(seedDataPath, "/net/events", "")
	if err != nil {
		log.Panicf("error get working directory: %s", err)
	}
	mountPath := seedDataPath + "/storages/assets/db/sp.sql"

	req := testcontainers.ContainerRequest{
		Image:        "mysql:latest",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": pw,
			"MYSQL_DATABASE":      db,
		},
		BindMounts: map[string]string{
			mountPath: "/docker-entrypoint-initdb.d/sp.sql",
		},
		WaitingFor: wait.ForLog("port: 3306  MySQL Community Server - GPL"),
	}

	mysqlC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		log.Panicf("error starting mysql container: %s", err)
	}

	closeFunc = func() {
		log.Println("terminating container")

		if err = mysqlC.Terminate(ctx); err != nil {
			log.Panicf("error terminating mysql container: %s", err)
		}
	}

	host, _ = mysqlC.Host(ctx)
	p, _ := mysqlC.MappedPort(ctx, "3306/tcp")
	port = uint16(p.Int())

	return
}

func createTestRedisContainer(config *cache.Config) (closeFunc func()) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "redis:latest",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}
	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Panic(err)
	}

	closeFunc = func() {
		_ = redisC.Terminate(ctx)
	}

	config.Host, _ = redisC.Host(ctx)
	p, _ := redisC.MappedPort(ctx, "6379/tcp")
	config.Port = p.Port()

	return
}

func SendMessageToMock(spAddress string, cmd string, pb proto.Message) {

	SPConn := client.NewClient(spAddress, false)

	d, err := proto.Marshal(pb)
	if err != nil {
		log.Panicf("proto marshal error: %v", err)
	}

	m := &msg.RelayMsgBuf{
		MSGHead: header.MakeMessageHeader(1, 1, uint32(len(d)), cmd),
		MSGData: d,
	}

	_ = SPConn.Write(m)
}
