package network

import (
	"context"
	"time"

	"github.com/stratosnet/sds/utils"
)

func (p *Network) ScheduleReloadPPStatus(ctx context.Context, future time.Duration) {
	utils.DebugLog("scheduled to get pp status from sp after: ", future.Seconds(), "second")
	p.ppPeerClock.AddJobWithInterval(future, p.GetPPStatusInitPPList(ctx))
}
