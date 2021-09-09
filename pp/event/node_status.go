package event

import (
	"fmt"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"

	"github.com/stratosnet/sds/msg/header"
)

// ReportNodeStatus
func ReportNodeStatus() {
	if setting.IsStartMining {
		go doReportNodeStatus()
	}
}

func doReportNodeStatus() {
	rnsReq, err := reqNodeStatusData()
	if err != nil {
		utils.ErrorLog("Couldn't build PP RNS request: " + err.Error())
		return
	}
	fmt.Println("Sending RNS message to SP! " + rnsReq.String())
	SendMessageToSPServer(rnsReq, header.ReqReportNodeStatus)
}
