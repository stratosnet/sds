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
	go StartListenServer(setting.Config.Port)
	GetIndexNodeList()
	GetPPStatusInitPPList()
	//go SendLatencyCheckMessageToIndexNodeList()
	StartStatusReportToIndexNode()
	ListenOffline()
}

func InitPeer(registerFn func()) {
	// TODO: To make sure this InitPeer method is correctly called and work as expected
	utils.DebugLog("InitPeer")
	//todo: register func call shouldn't be in peers package
	registerFn()
	GetIndexNodeList()
	GetPPStatusInitPPList()
	//go SendLatencyCheckMessageToIndexNodeList()
	go ListenOffline()
}

func RegisterToIndexNode(toIndexNode bool) {
	if toIndexNode {
		SendMessageToIndexNodeServer(requests.ReqRegisterData(), header.ReqRegister)
		utils.Log("SendMessage(conn, req, header.ReqRegister) to Index Node")
	} else {
		SendMessage(client.PPConn, requests.ReqRegisterData(), header.ReqRegister)
		utils.Log("SendMessage(conn, req, header.ReqRegister) to Index Node")
	}
}

func StartMining() {
	if setting.CheckLogin() {
		if setting.IsPP && !setting.IsLoginToIndexNode {
			utils.DebugLog("Bond to Index Node and start mining")
			SendMessageToIndexNodeServer(requests.ReqRegisterData(), header.ReqRegister)
		} else if setting.IsPP && !setting.IsStartMining {
			utils.DebugLog("Sending ReqMining message to Index Node")
			SendMessageToIndexNodeServer(requests.ReqMiningData(), header.ReqMining)
		} else if setting.IsStartMining {
			utils.Log("mining already started")
		} else {
			utils.Log("register as miner first")
		}
	}
}
