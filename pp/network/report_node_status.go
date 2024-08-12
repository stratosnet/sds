package network

import (
	"context"
	"sync"
	"time"

	"github.com/stratosnet/sds/framework/msg/header"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
)

var (
	nodeStatusResponded     = false
	consecutiveFailureCount = 0
	nodeStatusMutex         sync.Mutex
)

const (
	MAX_NUMBER_NODE_STATUS_FALIURE = 3
)

// StartStatusReportToSP to start a timer scheduling reporting Node Status to SP
func (p *Network) StartStatusReportToSP(ctx context.Context) {
	utils.DebugLog("Status will be reported to SP while mining")
	// trigger consecutive reports with interval
	p.ppPeerClock.AddJobRepeat(time.Second*setting.NodeReportIntervalSec, 0, p.doReportNodeStatus(ctx))
}

func (p *Network) doReportNodeStatus(ctx context.Context) func() {
	return func() {
		nodeStatusMutex.Lock()
		defer nodeStatusMutex.Unlock()
		if !nodeStatusResponded {
			if consecutiveFailureCount > MAX_NUMBER_NODE_STATUS_FALIURE {
				go p.ChangeSp(ctx)
				return
			}
			consecutiveFailureCount++
		} else {
			consecutiveFailureCount = 0
		}

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

func (p *Network) NodeStatusResponded(ctx context.Context) {
	nodeStatusMutex.Lock()
	defer nodeStatusMutex.Unlock()
	nodeStatusResponded = true
}
