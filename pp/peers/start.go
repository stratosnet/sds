package peers

import (
	"context"
	"path/filepath"

	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

func StartPP(ctx context.Context, registerFn func()) {
	GetNetworkAddress()
	peerList.Init(setting.NetworkAddress, filepath.Join(setting.Config.PPListDir, "pp-list"))
	//todo: register func call shouldn't be in peers package
	registerFn()
	go StartListenServer(ctx, setting.Config.Port)
	GetSPList(ctx)()
	GetPPStatusInitPPList(ctx)()
	//go SendLatencyCheckMessageToSPList()
	StartPpLatencyCheck(ctx)
	StartStatusReportToSP(ctx)
	go ListenOffline(ctx)
}

func InitPeer(ctx context.Context, registerFn func()) {
	// TODO: To make sure this InitPeer method is correctly called and work as expected
	utils.DebugLog("InitPeer")
	//todo: register func call shouldn't be in peers package
	registerFn()
	GetSPList(ctx)()
	GetPPStatusInitPPList(ctx)()
	//go SendLatencyCheckMessageToSPList()
	go ListenOffline(ctx)
}

func RegisterToSP(ctx context.Context, toSP bool) {
	if toSP {
		SendMessageToSPServer(ctx, requests.ReqRegisterData(), header.ReqRegister)
		pp.Log(ctx, "SendMessage(conn, req, header.ReqRegister) to SP")
	} else {
		SendMessage(ctx, client.PPConn, requests.ReqRegisterData(), header.ReqRegister)
		pp.Log(ctx, "SendMessage(conn, req, header.ReqRegister) to PP")
	}
}

func StartMining(ctx context.Context) {
	if setting.CheckLogin() {
		if setting.IsPP && !setting.IsLoginToSP {
			pp.DebugLog(ctx, "Bond to SP and start mining")
			SendMessageToSPServer(ctx, requests.ReqRegisterData(), header.ReqRegister)
		} else if setting.IsPP && !setting.IsStartMining {
			utils.DebugLog("Sending ReqMining message to SP")
			SendMessageToSPServer(ctx, requests.ReqMiningData(), header.ReqMining)
		} else if setting.IsStartMining {
			pp.Log(ctx, "mining already started")
		} else {
			pp.Log(ctx, "register as miner first")
		}
	}
}
