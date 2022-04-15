package sds

import (
	"net"

	setting "github.com/stratosnet/sds/cmd/relayd/config"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/utils"
)

func NewClient(server string) *cf.ClientConn {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", server)
	if err != nil {
		utils.ErrorLog("Couldn't resolve TCP address", err)
		return nil
	}
	c, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		utils.ErrorLog("DialTCP failed for SDS", err)
		return nil
	}
	onMessage := cf.OnMessageOption(func(msg msg.RelayMsgBuf, c core.WriteCloser) {})
	bufferSize := cf.BufferSizeOption(100)
	logOpen := cf.LogOpenOption(true)
	minAppVer := cf.MinAppVersionOption(setting.Config.Version.MinAppVer)
	options := []cf.ClientOption{
		onMessage,
		bufferSize,
		logOpen,
		cf.ReconnectOption(),
		minAppVer,
	}
	conn := cf.CreateClientConn(0, c, options...)
	conn.Start()

	return conn
}
