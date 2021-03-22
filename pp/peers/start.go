package peers

import (
	"github.com/qsnetwork/qsds/pp/client"
	"github.com/qsnetwork/qsds/pp/event"
	"github.com/qsnetwork/qsds/pp/setting"
	"github.com/qsnetwork/qsds/utils"
)

// Start Start
func Start(isPP bool) {
	GetWalletAddress()
	GetNetwrokAddress()
	event.RegisterEventHandle()
	if !isPP {
		initPPList()
	} else {
		client.SPConn = client.NewClient(setting.Config.SPNetAddress, true)
	}
	initBPList()
}

// StartPP StartPP
func StartPP() {
	GetWalletAddress()
	GetNetwrokAddress()
	event.RegisterEventHandle()
	// client.SPConn = client.NewClient(setting.Config.SPNetAddress, true)
	initPPList()
	initBPList()
	go listenOffline()
}

// InitPeer InitPeer
func InitPeer() {

	utils.DebugLog("InitPeer InitPeerInitPeer InitPeerInitPeer InitPeer")
	event.RegisterEventHandle()
	initPPList()
	initBPList()
	go listenOffline()
}
