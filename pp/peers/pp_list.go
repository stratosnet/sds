package peers

import (
	"sync"
	"time"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
)

// InitPPList
func InitPPList() {
	pplist := setting.GetLocalPPList()
	if len(pplist) == 0 {
		GetPPList()
	} else {
		for _, ppInfo := range pplist {
			client.PPConn = client.NewClient(ppInfo.NetworkAddress, true)
			if client.PPConn == nil {

				setting.DeletePPList(ppInfo.NetworkAddress)
			} else {
				RegisterChain(false)
				return
			}
		}

		GetPPList()
	}
}

func StartStatusReportToSP() {
	utils.DebugLog("Status will be reported to SP while mining")
	// trigger first report at time-0 immediately
	ReportNodeStatus()
	// trigger consecutive reports with interval
	clock := clock.NewClock()
	clock.AddJobRepeat(time.Second*setting.NodeReportIntervalSec, 0, ReportNodeStatus)
}

// GetPPList P node get PPList
func GetPPList() {
	utils.DebugLog("SendMessage(client.SPConn, req, header.ReqGetPPList)")
	SendMessageToSPServer(types.ReqGetPPlistData(), header.ReqGetPPList)
}

// RegisterPeerMap
var RegisterPeerMap = &sync.Map{} // make(map[string]int64)
