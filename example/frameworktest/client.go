package main

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/utils"
	"math/rand"
	"net"
	"time"
)

// counter number of clients in this test
var counter uint64 = 0

func NewClient() {
	// dail the server
	tcpAddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:55555")
	if err != nil {
		utils.ErrorLogf("resolve TCP address error: %v", err)
	}
	c, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		utils.ErrorLogf("connect failed", err)
		return
	}
	serverPortOpt := cf.ServerPortOption(55555)
	options := []cf.ClientOption{
		cf.BufferSizeOption(100),
		cf.LogOpenOption(true),
		serverPortOpt,
	}

	conn := cf.CreateClientConn(0, c, options...)
	conn.Start()

	counter++
	// loc_c identifier of this client
	loc_c := counter

	// message
	cmd := "RspBdVer"
	req := &protos.RspBadVersion{
		Version:        int32(7),
		MinimumVersion: int32(7),
		Command:        cmd,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		utils.ErrorLog(err)
		return
	}
	message := &msg.RelayMsgBuf{
		MSGHead: header.MakeMessageHeader(1, 7, uint32(len(data)), header.RspBadVersion, utils.ZeroId()),
		MSGData: data,
	}

	// random time to close this client
	ctime := rand.Intn(100)
	var done = make(chan bool)
	go func() {
		for {
			// random interval to send a message to the server
			t := rand.Intn(300)
			select {
			case <-time.After(time.Millisecond * time.Duration(10) * time.Duration(t+1)):
				conn.Write(message)

			case <-done:
				fmt.Println("close up the connection")
				conn.ClientClose()
				counter--
				return
			}
		}
	}()
	select {
	case <-time.After(time.Second * time.Duration(ctime+1)):
		// this condition is used for control which client to close
		if loc_c == 950 {
			done <- true
		}
	}
}

func main() {
	utils.NewDefaultLogger("./logs/client.log", true, true)

	for {
		select {
		// start a client every 50 ms up to 1000 clients
		case <-time.After(time.Millisecond * 5):
			if counter < 1000 {
				go NewClient()
			}
		}
	}
}
