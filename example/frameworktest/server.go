package main

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/utils"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"
)

var (
	dafualt_prof = "server.prof"
)

func main() {
	utils.NewDefaultLogger("./logs/server.log", true, true)

	var counter int64 = 0
	// pprof
	f, err := os.Create(dafualt_prof)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	signal.Notify(make(chan os.Signal), syscall.SIGPROF)
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
	defer pprof.StopCPUProfile()

	// server
	host := "0.0.0.0:55555"
	netListen, err := net.Listen("tcp", host)
	if err != nil {
		log.Fatal(err)
	}

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

	server := core.CreateServer(
		//core.OnConnectOption(server.OnConnectOption),
		//core.OnCloseOption(server.OnCloseOption),
		////core.MaxConnectionsOption(utils.MaxConnections),
		core.MaxConnectionsOption(12000),
		core.MaxFlowOption(125*1024*1024),
		//core.MinAppVersionOption(server.Conf.Version.MinAppVer),
		//core.P2pAddressOption(server.p2pAddress),
	)
	log.Print("start framework server")
	go server.Start(netListen)
	go func() {
		for {
			select {
			case <-time.After(time.Second * 5):
				fmt.Println("Num Goroutine:", runtime.NumGoroutine())
				fmt.Println("Counter:", core.Test_counter)
				server.Broadcast(message)
				counter++
			}
		}
	}()

	select {
	// set test duration to 10 min and gracefully return with profiler file well written
	case <-time.After(time.Minute * 10):
		return
	}
}
