package peers

import (
	"github.com/shirou/gopsutil/disk"

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

// GetDHInfo
func GetDHInfo() (uint64, uint64) {
	d, err := disk.Usage("/")
	if err != nil {
		utils.ErrorLog("GetDHInfo", err)
		return 0, 0
	}
	return d.Total, d.Free
}
