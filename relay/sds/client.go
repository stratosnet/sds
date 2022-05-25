package sds

import (
	"net"
	"strconv"

	setting "github.com/stratosnet/sds/cmd/relayd/config"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/types"
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

	p2pAddress, err := getClientP2pAddressOpt()
	if err != nil {
		utils.ErrorLog("Couldn't generate fake p2p address", err)
		return nil
	}

	serverPort, err := strconv.ParseUint(setting.Config.SDS.HandshakePort, 10, 16)
	if err != nil {
		utils.ErrorLogf("Invalid port number in config [%v]: %v", setting.Config.SDS.HandshakePort, err.Error())
		return nil
	}
	serverPortOpt := cf.ServerPortOption(uint16(serverPort))

	options := []cf.ClientOption{
		cf.OnMessageOption(func(msg msg.RelayMsgBuf, c core.WriteCloser) {}),
		cf.BufferSizeOption(100),
		cf.LogOpenOption(true),
		cf.ReconnectOption(),
		cf.MinAppVersionOption(setting.Config.Version.MinAppVer),
		p2pAddress,
		serverPortOpt,
	}
	conn := cf.CreateClientConn(0, c, options...)
	conn.Start()

	return conn
}

func NewTCPServer() (*core.Server, error) {
	p2pAddress, err := getServerP2pAddressOpt()
	if err != nil {
		return nil, err
	}
	server := core.CreateServer(
		core.BufferSizeOption(10000),
		core.MinAppVersionOption(setting.Config.Version.MinAppVer),
		p2pAddress,
	)
	return server, err
}

func getClientP2pAddressOpt() (cf.ClientOption, error) {
	p2pAddress, err := types.P2pAddressToBech(types.Address{})
	if err != nil {
		return nil, err
	}
	return cf.P2pAddressOption(p2pAddress), nil
}

func getServerP2pAddressOpt() (core.ServerOption, error) {
	p2pAddress, err := types.P2pAddressToBech(types.Address{})
	if err != nil {
		return nil, err
	}
	return core.P2pAddressOption(p2pAddress), nil
}
