package peers

import (
	"path/filepath"

	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

func StartPP(registerFn func()) {
	GetNetworkAddress()
	peerList.Init(setting.NetworkAddress, filepath.Join(setting.Config.PPListDir, "pp-list"))
	//todo: register func call shouldn't be in peers package
	registerFn()
	GetSPList()
	GetPPStatusFromSP()
	//go SendLatencyCheckMessageToSPList()
	//InitPPList() // moved to rsp of GetPPStatusFromSP()
	StartStatusReportToSP()
	ListenOffline()
}

func InitPeer(registerFn func()) {
	// TODO: To make sure this InitPeer method is correctly called and work as expected
	utils.DebugLog("InitPeer")
	//todo: register func call shouldn't be in peers package
	registerFn()
	GetSPList()
	GetPPStatusFromSP()
	//go SendLatencyCheckMessageToSPList()
	//InitPPList() // moved to rsp of GetPPStatusFromSP()
	go ListenOffline()
}

func RegisterToSP(toSP bool) {
	if toSP {
		SendMessageToSPServer(requests.ReqRegisterData(), header.ReqRegister)
		utils.Log("SendMessage(conn, req, header.ReqRegister) to SP")
	} else {
		SendMessage(client.PPConn, requests.ReqRegisterData(), header.ReqRegister)
		utils.Log("SendMessage(conn, req, header.ReqRegister) to PP")
	}
}

func StartMining() {
	if setting.CheckLogin() {
		if setting.IsPP && !setting.IsLoginToSP {
			utils.DebugLog("Bond to SP and start mining")
			SendMessageToSPServer(requests.ReqRegisterData(), header.ReqRegister)
		} else if setting.IsPP && !setting.IsStartMining {
			utils.DebugLog("Sending ReqMining message to SP")
			SendMessageToSPServer(requests.ReqMiningData(), header.ReqMining)
		} else if setting.IsStartMining {
			utils.Log("mining already started")
		} else {
			utils.Log("register as miner first")
		}
	}
}
