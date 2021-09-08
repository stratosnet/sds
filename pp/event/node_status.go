package event

import (
	//"context"
	"fmt"
	//"github.com/stratosnet/sds/framework/core"
	//"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/utils"
)

// ReportNodeStatus
func ReportNodeStatus() {
	rnsReq, err := reqNodeStatusData()
	if err != nil {
		utils.ErrorLog("Couldn't build PP RNS request: " + err.Error())
		return
	}
	fmt.Println("Sending RNS message to SP! " + rnsReq.String())
	SendMessageToSPServer(rnsReq, header.ReqReportNodeStatus)
}

//// RspNodeStatus
//func RspNodeStatus(ctx context.Context, conn core.WriteCloser) {
//	// utils.DebugLog("ResHeartBeat")
//	switch conn.(type) {
//	case *core.ServerConn:
//		msg := msg.RelayMsgBuf{
//			MSGHead: PPMsgHeader(nil, header.RspHeart),
//		}
//		conn.Write(&msg)
//	}
//}
