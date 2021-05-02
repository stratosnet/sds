package sds

import (
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/utils"
	"net"
)

func NewClient(server string) *cf.ClientConn {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", server)
	utils.CheckError(err)
	c, err := net.DialTCP("tcp", nil, tcpAddr)
	if utils.CheckError(err) {
		utils.DebugLog(server, "connect failed")
		return nil
	}
	utils.Log("connect success")
	onConnect := cf.OnConnectOption(func(c spbf.WriteCloser) bool {
		utils.DebugLog("on connect")
		return true
	})
	onError := cf.OnErrorOption(func(c spbf.WriteCloser) {
		utils.Log("on error")
	})
	onClose := cf.OnCloseOption(func(c spbf.WriteCloser) {
		utils.Log("on close", c.(*cf.ClientConn).GetName())
	})
	onMessage := cf.OnMessageOption(func(msg msg.RelayMsgBuf, c spbf.WriteCloser) {})
	bufferSize := cf.BufferSizeOption(100)
	logOpen := cf.LogOpenOption(true)
	options := []cf.ClientOption{
		onConnect,
		onError,
		onClose,
		onMessage,
		bufferSize,
		logOpen,
	}
	conn := cf.CreateClientConn(0, c, options...)
	conn.Start()

	return conn
}
