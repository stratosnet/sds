package peers

import (
	"net"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/utils"
)

//todo: pp server should be move out of peers package

// PPServer
type PPServer struct {
	*core.Server
}

var ppServ *PPServer
var ppPeerClock = clock.NewClock()

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
		netID := conn.(*core.ServerConn).GetNetID()
		peerList.PPDisconnectedNetId(netID)
	})
	bufferSize := core.BufferSizeOption(10000)
	return &PPServer{
		core.CreateServer(onConnectOption, onErrorOption, onCloseOption, bufferSize, core.LogOpenOption(true)),
	}
}
