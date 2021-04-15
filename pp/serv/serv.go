package serv

import (
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/utils"
	"net"
	"sync"
)

// PPServer
type PPServer struct {
	*spbf.Server
}

var ppServ *PPServer

// RegisterPeerMap
var RegisterPeerMap = &sync.Map{} // make(map[string]int64)

// GetPPServer
func GetPPServer() *PPServer {
	return ppServ
}

// StartListenServer
func StartListenServer(port string) {
	netListen, err := net.Listen("tcp4", port)
	if utils.CheckError(err) {
		utils.ErrorLog("StartListenServer", err)
	}
	spbServer := NewServer()
	ppServ = spbServer
	utils.Log("StartListenServer!!! ", port)
	err1 := spbServer.Start(netListen)
	if utils.CheckError(err1) {
		utils.ErrorLog("StartListenServer Error", err1)
	}
}

// NewServer returns a Server.
func NewServer() *PPServer {
	onConnectOption := spbf.OnConnectOption(func(conn spbf.WriteCloser) bool {
		utils.Log("on connect")
		return true
	})
	onErrorOption := spbf.OnErrorOption(func(conn spbf.WriteCloser) {
		utils.Log("on error")
	})
	onCloseOption := spbf.OnCloseOption(func(conn spbf.WriteCloser) {
		net := conn.(*spbf.ServerConn).GetName()
		netID := conn.(*spbf.ServerConn).GetNetID()
		removePeer(netID)
		utils.DebugLog(net, netID, "offline")
	})
	bufferSize := spbf.BufferSizeOption(10000)
	return &PPServer{
		spbf.CreateServer(onConnectOption, onErrorOption, onCloseOption, bufferSize),
	}
}

func removePeer(netID int64) {

	f := func(k, v interface{}) bool {
		if v == netID {
			RegisterPeerMap.Delete(k)
			return false
		}
		return true
	}
	RegisterPeerMap.Range(f)
}
