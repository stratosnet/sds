package main

import (
	"fmt"
	"net"
	"github.com/qsnetwork/sds/framework/client/cf"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg"
	"github.com/qsnetwork/sds/msg/header"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/utils"
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
		w := "12345678901234567890123456789012345" + no
		n := "localhost:" + no

		c := newCONN()

		c.login(w, n)
		c.registerPP(w)
		c.login(w, n)

		ppList[ppNum-1] = c

		ppNum++
	})

	select {}
}

func newCONN() *CONN {

	tcpAddr, err := net.ResolveTCPAddr("tcp4", "localhost:8888")
	utils.CheckError(err)
	c, err := net.DialTCP("tcp", nil, tcpAddr)
	if utils.CheckError(err) {
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
	if utils.CheckError(err) {
		panic(err)
	}
	msg := &msg.RelayMsgBuf{
		MSGHead: header.MakeMessageHeader(1, 1, uint32(len(data)), cmd),
		MSGData: data,
	}
	c.Conn.Write(msg)
}

func (c *CONN) pplist() {
	c.send(&protos.ReqGetPPList{
		MyAddress: &protos.PPBaseInfo{
			WalletAddress:  "",
			NetworkAddress: "",
		},
	}, header.ReqGetPPList)
}

func (c *CONN) login(w, n string) {
	c.send(&protos.ReqRegister{
		Address: &protos.PPBaseInfo{
			WalletAddress:  w,
			NetworkAddress: n,
		},
		PublicKey: w + ":publicKey",
	}, header.ReqRegister)
}

func (c *CONN) registerPP(w string) {
	c.send(&protos.ReqRegisterNewPP{
		WalletAddress: w,
		DiskSize:      1024 * 1024 * 1024 * 720,
		MemorySize:    1024 * 1024 * 1024 * 2,
		OsAndVer:      "CentOS 7",
		CpuInfo:       "intel i7",
		MacAddress:    "12345678901234567",
		Version:       1,
		PubKey:        w + ":publicKey",
	}, header.ReqRegisterNewPP)
}
