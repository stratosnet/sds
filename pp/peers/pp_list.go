package peers

import (
	"time"

	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
)

// Peers is a list of the know PP node peers
var Peers types.PeerList

const (
	RELOAD_PP_LIST_INTERVAL_SHORT  = 5 * time.Second
	RELOAD_PP_LIST_INTERVAL_MEDIUM = 15 * time.Second
	RELOAD_PP_LIST_INTERVAL_LONG   = 30 * time.Second
)

// InitPPList
func InitPPList() {
	pplist := Peers.GetPPList()
	if len(pplist) == 0 {
		GetPPList()
	} else {
		if success := ConnectToGatewayPP(pplist); !success {
			GetPPList()
			return
		}
		if setting.IsAuto && setting.State == types.PP_ACTIVE && !setting.IsLoginToSP {
			RegisterToSP(true)
		}
	}
}

func StartStatusReportToSP() {
	utils.DebugLog("Status will be reported to SP while mining")
	// trigger first report at time-0 immediately
	ReportNodeStatus()
	// trigger consecutive reports with interval
	ppPeerClock.AddJobRepeat(time.Second*setting.NodeReportIntervalSec, 0, ReportNodeStatus)
}

// GetPPList P node get ppList from sp
func GetPPList() {
	utils.DebugLog("SendMessage(client.SPConn, req, header.ReqGetPPList)")
	SendMessageToSPServer(requests.ReqGetPPlistData(), header.ReqGetPPList)
}

func ConnectToGatewayPP(pplist []*types.PeerInfo) bool {
	for _, ppInfo := range pplist {
		if ppInfo.NetworkAddress == setting.NetworkAddress {
			Peers.DeletePPByNetworkAddress(ppInfo.NetworkAddress)
			continue
		}
		client.PPConn = client.NewClient(ppInfo.NetworkAddress, true)
		if client.PPConn != nil {
			return true
		}
		utils.DebugLog("failed to conn PP，delete:", ppInfo)
		Peers.DeletePPByNetworkAddress(ppInfo.NetworkAddress)
	}
	return false
}

/**
Long: 	pp not activated
Medium: mining not yet started
Short: 	by default (mining)
*/
func ScheduleReloadPPlist() {
	var future time.Duration
	if setting.State != types.PP_ACTIVE {
		future = RELOAD_PP_LIST_INTERVAL_LONG
	} else if !setting.IsStartMining {
		future = RELOAD_PP_LIST_INTERVAL_MEDIUM
	} else {
		future = RELOAD_PP_LIST_INTERVAL_SHORT
	}
	utils.DebugLog("scheduled to get pp-list after: ", future.Seconds(), "second")
	ppPeerClock.AddJobWithInterval(future, GetPPList)
}
