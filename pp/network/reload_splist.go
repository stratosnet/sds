package network

import (
	"context"
	"time"

	"github.com/stratosnet/sds/utils"
)

func (p *Network) ScheduleReloadSPlist(ctx context.Context, future time.Duration) {
	utils.DebugLog("scheduled to get sp-list after: ", future.Seconds(), "second")
	// scheduled report should only be sent when it's registered
	if state := p.GetStateFromFsm(); state.Id == STATE_INIT {
		p.ppPeerClock.AddJobWithInterval(future, p.GetSPList(ctx))
	}
}
