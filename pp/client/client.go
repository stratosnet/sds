package client

import (
	"net"
	"strconv"
	"sync"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// Offline Offline
type Offline struct {
	IsIndexNode    bool
	NetworkAddress string
}

// OfflineChan OfflineChan
var OfflineChan = make(chan *Offline, 2)

// IndexNodeConn super node connection
var IndexNodeConn *cf.ClientConn

// PPConn current connected pp node
var PPConn *cf.ClientConn

// UpConnMap upload connection
var UpConnMap = &sync.Map{}

// DownloadConnMap  download connection between  P-PP
var DownloadConnMap = &sync.Map{}

// ConnMap client connection map
var ConnMap = make(map[string]*cf.ClientConn)

// NewClient
func NewClient(server string, heartbeat bool) *cf.ClientConn {

	tcpAddr, err := net.ResolveTCPAddr("tcp4", server)
	if err != nil {
		utils.ErrorLogf("resolve TCP address error: %v", err)
	}
	c, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		utils.DebugLog(server, "connect failed", err)
		return nil
	}
	utils.Log("connect success")
	onConnect := cf.OnConnectOption(func(c core.WriteCloser) bool {
		utils.DebugLog("on connect")
		return true
	})
	onError := cf.OnErrorOption(func(c core.WriteCloser) {
		utils.Log("on error")
	})
	onClose := cf.OnCloseOption(func(c core.WriteCloser) {
		utils.Log("on close", c.(*cf.ClientConn).GetName())
		delete(ConnMap, c.(*cf.ClientConn).GetName())

		if PPConn != nil {
			if PPConn == c.(*cf.ClientConn) {
				utils.DebugLog("lost gateway PP conn, delete and change to new PP")
				select {
				case OfflineChan <- &Offline{
					IsIndexNode:    false,
					NetworkAddress: PPConn.GetRemoteAddr(),
				}:
				default:
					break
				}
			}
		}
		if IndexNodeConn != nil {
			if IndexNodeConn.GetName() == c.(*cf.ClientConn).GetName() {
				utils.DebugLog("lost SP conn, name: ", IndexNodeConn.GetName(), " netId is ", IndexNodeConn.GetNetID())
				IndexNodeConn = nil
				select {
				case OfflineChan <- &Offline{
					IsIndexNode: true,
				}:
				default:
					break
				}
			}
		}

	})

	serverPort, err := strconv.ParseUint(setting.Config.Port, 10, 16)
	if err != nil {
		utils.ErrorLogf("Invalid port number in config [%v]: %v", setting.Config.Port, err.Error())
		return nil
	}
	serverPortOpt := cf.ServerPortOption(uint16(serverPort))

	options := []cf.ClientOption{
		onConnect,
		onError,
		onClose,
		cf.OnMessageOption(func(msg msg.RelayMsgBuf, c core.WriteCloser) {}),
		cf.BufferSizeOption(100),
		cf.HeartCloseOption(!heartbeat),
		cf.LogOpenOption(true),
		cf.MinAppVersionOption(setting.Config.Version.MinAppVer),
		cf.P2pAddressOption(setting.P2PAddress),
		serverPortOpt,
	}
	conn := cf.CreateClientConn(0, c, options...)
	conn.Start()
	ConnMap[server] = conn
	return conn
}

func GetConnectionName(conn core.WriteCloser) string {
	if conn == nil {
		return ""
	}
	switch conn.(type) {
	case *core.ServerConn:
		return conn.(*core.ServerConn).GetName()
	case *cf.ClientConn:
		return conn.(*cf.ClientConn).GetName()
	}
	return ""
}
