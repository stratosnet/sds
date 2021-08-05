package sds

import (
	"fmt"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"net"
)

func NewClient(server string) *cf.ClientConn {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", server)
	if err != nil {
		fmt.Println("Couldn't resolve TCP address: " + err.Error())
		return nil
	}
	c, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		fmt.Println("DialTCP failed for SDS: " + err.Error())
		return nil
	}

	onMessage := cf.OnMessageOption(func(msg msg.RelayMsgBuf, c core.WriteCloser) {})
	bufferSize := cf.BufferSizeOption(100)
	logOpen := cf.LogOpenOption(true)
	options := []cf.ClientOption{
		onMessage,
		bufferSize,
		logOpen,
		cf.ReconnectOption(),
	}
	conn := cf.CreateClientConn(0, c, options...)
	conn.Start()

	return conn
}
