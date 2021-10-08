package peers

import (
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
)

// StartPP
func StartPP(registerFn func()) {
	GetNetworkAddress()
	//todo: register func call shouldn't be in peers package
	registerFn()
	GetSPList()
	InitPPList()
	ListenOffline()
	StartStatusReportToSP()
}

// InitPeer
func InitPeer(registerFn func()) {
	utils.DebugLog("InitPeer InitPeerInitPeer InitPeerInitPeer InitPeer")
	//todo: register func call shouldn't be in peers package
	registerFn()
	GetSPList()
	InitPPList()
	go ListenOffline()
}

// RegisterChain
func RegisterChain(toSP bool) {
	if toSP {
		SendMessageToSPServer(types.ReqRegisterData(), header.ReqRegister)
		utils.Log("SendMessage(conn, req, header.ReqRegister) to SP")
	} else {
		SendMessage(client.PPConn, types.ReqRegisterData(), header.ReqRegister)
		utils.Log("SendMessage(conn, req, header.ReqRegister) to PP")
	}

}

// StartMining
func StartMining() {
	if setting.CheckLogin() {
		if setting.IsPP {
			utils.DebugLog("Sending ReqMining message to SP")
			SendMessageToSPServer(types.ReqMiningData(), header.ReqMining)
		} else {
			utils.Log("register as miner first")
		}
	}
}
