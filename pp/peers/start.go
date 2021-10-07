package peers

import (
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/utils"
)

// StartPP
func StartPP() {
	err := GetWalletAddress()
	if err != nil {
		utils.ErrorLog(err)
		return
	}
	GetNetworkAddress()
	event.RegisterEventHandle()
	event.GetSPList()
	initPPList()
	listenOffline()
	startStatusReportToSP()
}

// InitPeer
func InitPeer() {
	utils.DebugLog("InitPeer InitPeerInitPeer InitPeerInitPeer InitPeer")
	event.RegisterEventHandle()
	event.GetSPList()
	initPPList()
	go listenOffline()
}
