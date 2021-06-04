package main

import (
	"fmt"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/utils"
	"net"
	"time"

	"github.com/alex023/clock"

	"github.com/golang/protobuf/proto"
)

const MAX_PP_NUM = 10

func main() {

	ppNum := 1
	ppList := make([]*CONN, MAX_PP_NUM)

	main := clock.NewClock()
	main.AddJobRepeat(time.Second*1, uint64(cap(ppList)), func() {

		no := fmt.Sprintf("%07d", ppNum)
		ppBaseInfo := &protos.PPBaseInfo{
			WalletAddress: "12345678901234567890123456789012345" + no,
			NetworkId: &protos.NetworkId{
				PublicKey:      "04985aab77ca25ccedda4ae680dc3f05c56830a67fe9e6570ec6702c00652382940ba7fe7da83f7a31e954f6bd46b9d7ed53bdbe59079ef84b8114576609fa3f8d",
				NetworkAddress: "localhost:" + no,
			},
		}

		c := newCONN()

		c.login(ppBaseInfo)
		c.registerPP(ppBaseInfo)
		c.login(ppBaseInfo)

		ppList[ppNum-1] = c

		ppNum++
	})

	select {}
}

func newCONN() *CONN {

	tcpAddr, err := net.ResolveTCPAddr("tcp4", "localhost:8888")
	if err != nil {
		utils.ErrorLogf("new connection error : %v", err)
	}
	var c *net.TCPConn
	c, err = net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		panic(err)
	}
	onConnect := cf.OnConnectOption(func(c spbf.WriteCloser) bool {
		utils.Log("on connect")
		return true
	})
	onError := cf.OnErrorOption(func(c spbf.WriteCloser) {
		utils.Log("on error")
	})
	onClose := cf.OnCloseOption(func(c spbf.WriteCloser) {
		utils.Log("on close")
	})
	onMessage := cf.OnMessageOption(func(msg msg.RelayMsgBuf, c spbf.WriteCloser) {
	})

	bufferSize := cf.BufferSizeOption(1000)
	options := []cf.ClientOption{
		onConnect,
		onError,
		onClose,
		onMessage,
		bufferSize,
	}
	conn := &CONN{
		Conn: cf.CreateClientConn(0, c, options...),
	}
	conn.Conn.Start()
	return conn
}

type CONN struct {
	Conn *cf.ClientConn
}

func (c *CONN) send(message proto.Message, cmd string) {
	data, err := proto.Marshal(message)
	if err != nil {
		panic(err)
	}
	msg := &msg.RelayMsgBuf{
		MSGHead: header.MakeMessageHeader(1, 1, uint32(len(data)), cmd),
		MSGData: data,
	}
	c.Conn.Write(msg)
}

func (c *CONN) login(ppBaseInfo *protos.PPBaseInfo) {
	c.send(&protos.ReqRegister{
		Address: ppBaseInfo,
	}, header.ReqRegister)
}

func (c *CONN) registerPP(ppBaseInfo *protos.PPBaseInfo) {
	c.send(&protos.ReqRegisterNewPP{
		PpBaseInfo: ppBaseInfo,
		DiskSize:      1024 * 1024 * 1024 * 720,
		MemorySize:    1024 * 1024 * 1024 * 2,
		OsAndVer:      "CentOS 7",
		CpuInfo:       "intel i7",
		MacAddress:    "12345678901234567",
		Version:       1,
	}, header.ReqRegisterNewPP)
}
