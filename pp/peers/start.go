package peers

import (
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
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
