package network

import (
	"context"
	"time"

	"github.com/stratosnet/sds/framework/msg/header"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
)

// StartStatusReportToSP to start a timer scheduling reporting Node Status to SP
func (p *Network) StartStatusReportToSP(ctx context.Context) {
	utils.DebugLog("Status will be reported to SP while mining")
	// trigger consecutive reports with interval
	p.ppPeerClock.AddJobRepeat(time.Second*setting.NodeReportIntervalSec, 0, p.doReportNodeStatus(ctx))
}

func (p *Network) doReportNodeStatus(ctx context.Context) func() {
	return func() {
		// scheduled report should only be sent when it's registered
		if state := p.GetStateFromFsm(); state.Id == STATE_REGISTERED {
			go p.ReportNodeStatus(ctx)
		}
	}
}

func (p *Network) ReportNodeStatus(ctx context.Context) {
	status := requests.ReqNodeStatusData(p2pserver.GetP2pServer(ctx).GetP2PAddress().String())
	pp.DebugLog(ctx, "Sending RNS message to SP! "+status.String())

	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, status, header.ReqReportNodeStatus)
}
