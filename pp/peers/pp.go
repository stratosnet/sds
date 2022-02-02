package peers

import (
	"net"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/utils"
)

//todo: pp server should be move out of peers package

// PPServer
type PPServer struct {
	*core.Server
}

var ppServ *PPServer

// GetPPServer
func GetPPServer() *PPServer {
	return ppServ
}

func SetPPServer(pp *PPServer) {
	ppServ = pp
}

// StartListenServer
func StartListenServer(port string) {
	netListen, err := net.Listen("tcp4", ":"+port)
	if err != nil {
		utils.ErrorLog("StartListenServer", err)
	}
	spbServer := NewServer()
	ppServ = spbServer
	utils.Log("StartListenServer!!! ", port)
	err = spbServer.Start(netListen)
	if err != nil {
		utils.ErrorLog("StartListenServer Error", err)
	}
}

// NewServer returns a server.
func NewServer() *PPServer {
	onConnectOption := core.OnConnectOption(func(conn core.WriteCloser) bool {
		utils.Log("on connect")
		return true
	})
	onErrorOption := core.OnErrorOption(func(conn core.WriteCloser) {
		utils.Log("on error")
	})
	onCloseOption := core.OnCloseOption(func(conn core.WriteCloser) {
		net := conn.(*core.ServerConn).GetName()
		netID := conn.(*core.ServerConn).GetNetID()
		removePeer(netID)
		utils.DebugLog(net, netID, "offline")
	})
	bufferSize := core.BufferSizeOption(10000)
	return &PPServer{
		core.CreateServer(onConnectOption, onErrorOption, onCloseOption, bufferSize),
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
