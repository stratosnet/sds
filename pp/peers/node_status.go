package peers

import (
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
)

// ReportNodeStatus
func ReportNodeStatus() {
	if setting.IsStartMining && setting.State == types.PP_ACTIVE {
		status := requests.ReqNodeStatusData()
		go doReportNodeStatus(status)
	}
}

func doReportNodeStatus(status *protos.ReqReportNodeStatus) {
	utils.DebugLog("Sending RNS message to Index Node! " + status.String())
	SendMessageToIndexNodeServer(status, header.ReqReportNodeStatus)
	// if current reachable is too less, try refresh the list
	_, total, _ := peerList.GetPPList()
	if total > 0 && total <= 2 {
		GetPPListFromIndexNode()
	}
}
