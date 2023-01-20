package network

import (
	"context"
	"time"

	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// StartStatusReportToSP to start a timer scheduling reporting Node Status to SP
func (p *Network) StartStatusReportToSP(ctx context.Context) {
	utils.DebugLog("Status will be reported to SP while mining")
	// trigger first report at time-0 immediately
	p.ReportNodeStatus(ctx)()
	// trigger consecutive reports with interval
	p.ppPeerClock.AddJobRepeat(time.Second*setting.NodeReportIntervalSec, 0, p.ReportNodeStatus(ctx))
}

// ReportNodeStatus
func (p *Network) ReportNodeStatus(ctx context.Context) func() {
	return func() {
		if state := p.GetStateFromFsm(); state.Id == STATE_REGISTERING || state.Id == STATE_REGISTERED {
			status := requests.ReqNodeStatusData()
			go p.doReportNodeStatus(ctx, status)
		}
	}
}

// doReportNodeStatus
func (p *Network) doReportNodeStatus(ctx context.Context, status *protos.ReqReportNodeStatus) {
	pp.DebugLog(ctx, "Sending RNS message to SP! "+status.String())
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, status, header.ReqReportNodeStatus)
	// if current reachable is too less, try refresh the list
	_, total, connected := p2pserver.GetP2pServer(ctx).GetPPList(ctx)
	pp.Logf(ctx, "#pp_in_list:[%d], #pp_connected:[%d]", total, connected)
	if total > 0 && total <= 2 {
		p.GetPPListFromSP(ctx)
	}
}
