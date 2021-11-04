package peers

import (
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// ReportNodeStatus
func ReportNodeStatus() {
	if setting.IsStartMining {
		status := requests.ReqNodeStatusData()
		go doReportNodeStatus(status)
	}
}

func doReportNodeStatus(status *protos.ReqReportNodeStatus) {
	utils.DebugLog("Sending RNS message to SP! " + status.String())
	SendMessageToSPServer(status, header.ReqReportNodeStatus)
}
