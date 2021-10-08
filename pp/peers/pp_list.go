package peers

import (
	"github.com/stratosnet/sds/msg/protos"
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
		if success := SendRegisterRequestViaPP(pplist); !success {
			GetPPList()
		}
	}
}

func StartStatusReportToSP() {
	utils.DebugLog("Status will be reported to SP while mining")
	clock := clock.NewClock()
	clock.AddJobRepeat(time.Minute*5, 0, ReportNodeStatus)
}

// GetPPList P node get PPList
func GetPPList() {
	utils.DebugLog("SendMessage(client.SPConn, req, header.ReqGetPPList)")
	SendMessageToSPServer(types.ReqGetPPlistData(), header.ReqGetPPList)
}

func SendRegisterRequestViaPP(pplist []*protos.PPBaseInfo) bool {
	for _, ppInfo := range pplist {
		if ppInfo.NetworkAddress == setting.NetworkAddress {
			continue
		}
		client.PPConn = client.NewClient(ppInfo.NetworkAddress, true)
		if client.PPConn != nil {
			RegisterChain(false)
			return true
		}
		utils.DebugLog("failed to conn PP，delete:", ppInfo)
		setting.DeletePPList(ppInfo.NetworkAddress)
	}
	return false
}

// RegisterPeerMap
var RegisterPeerMap = &sync.Map{} // make(map[string]int64)
