package peers

import (
	"time"

	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
)

// PeerList is a list of the know PP node peers
var peerList types.PeerList

const (
	RELOAD_PP_LIST_INTERVAL_SHORT  = 5 * time.Second
	RELOAD_PP_LIST_INTERVAL_MEDIUM = 15 * time.Second
	RELOAD_PP_LIST_INTERVAL_LONG   = 30 * time.Second
)

// InitPPList
func InitPPList() {
	pplist, _, _ := peerList.GetPPList()
	if len(pplist) == 0 {
		GetPPListFromSP()
	} else {
		if success := ConnectToGatewayPP(pplist); !success {
			GetPPListFromSP()
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

// GetPPListFromSP node get ppList from sp
func GetPPListFromSP() {
	utils.DebugLog("SendMessage(client.SPConn, req, header.ReqGetPPList)")
	SendMessageToSPServer(requests.ReqGetPPlistData(), header.ReqGetPPList)
}

func ConnectToGatewayPP(pplist []*types.PeerInfo) bool {
	for _, ppInfo := range pplist {
		if ppInfo.NetworkAddress == setting.NetworkAddress {
			peerList.DeletePPByNetworkAddress(ppInfo.NetworkAddress)
			continue
		}
		client.PPConn = client.NewClient(ppInfo.NetworkAddress, true)
		if client.PPConn != nil {
			return true
		}
		utils.DebugLog("failed to conn PP，delete:", ppInfo)
		peerList.DeletePPByNetworkAddress(ppInfo.NetworkAddress)
	}
	return false
}

//ScheduleReloadPPlist
//	Long: 	pp not activated
//	Medium: mining not yet started
//	Short: 	by default (mining)
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
	ppPeerClock.AddJobWithInterval(future, GetPPListFromSP)
}

//GetPPList will just get the list from
func GetPPList() (list []*types.PeerInfo, total int64) {
	list, total, _ = peerList.GetPPList()
	return
}

//SavePPList will save the target list to local list
func SavePPList(target *protos.RspGetPPList) error {
	return peerList.SavePPList(target)
}

//UpdatePP will update one pp info to local list
func UpdatePP(pp *types.PeerInfo) {
	peerList.UpdatePP(pp)
}
