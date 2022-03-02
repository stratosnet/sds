package peers

import (
	"path/filepath"

	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// StartPP
func StartPP(registerFn func()) {
	GetNetworkAddress()
	Peers.Init(setting.NetworkAddress, filepath.Join(setting.Config.PPListDir, "pp-list"))
	//todo: register func call shouldn't be in peers package
	registerFn()
	GetSPList()
	GetPPStatusFromSP()
	//go SendLatencyCheckMessageToSPList()
	//InitPPList() // moved to rsp of GetPPStatusFromSP()
	ListenOffline()
	StartStatusReportToSP()
}

// InitPeer
func InitPeer(registerFn func()) {
	utils.DebugLog("InitPeer InitPeerInitPeer InitPeerInitPeer InitPeer")
	//todo: register func call shouldn't be in peers package
	registerFn()
	GetSPList()
	GetPPStatusFromSP()
	//go SendLatencyCheckMessageToSPList()
	//InitPPList() // moved to rsp of GetPPStatusFromSP()
	go ListenOffline()
}

// RegisterToSP
func RegisterToSP(toSP bool) {
	if toSP {
		SendMessageToSPServer(requests.ReqRegisterData(), header.ReqRegister)
		utils.Log("SendMessage(conn, req, header.ReqRegister) to SP")
	} else {
		SendMessage(client.PPConn, requests.ReqRegisterData(), header.ReqRegister)
		utils.Log("SendMessage(conn, req, header.ReqRegister) to PP")
	}
}

// StartMining
func StartMining() {
	if setting.CheckLogin() {
		if setting.IsPP {
			utils.DebugLog("Sending ReqMining message to SP")
			SendMessageToSPServer(requests.ReqMiningData(), header.ReqMining)
		} else {
			utils.Log("register as miner first")
		}
	}
}
