package network

import (
	"context"
	"math/rand"
	"time"

	"github.com/stratosnet/sds/utils"
)

const (
	SPLIST_INTERVAL_BASE = 60 // In second
	SPLIST_MAX_JITTER    = 10 // In second
	MAX_RETRY_TIMES      = 3
	ERROR_CODE_NO_SPLIST = -2
)

var (
	retryCounter int = 0
)

func (p *Network) StartGetSPList(ctx context.Context) func() {
	return func() {
		if state := p.GetStateFromFsm(); state.Id == STATE_INIT {
			retryCounter++
			if retryCounter >= MAX_RETRY_TIMES {
				utils.FatalLogfAndExit(ERROR_CODE_NO_SPLIST, "Fatal error: failed getting SP list, quit.")
			}
			p.GetSPList(ctx)()

			duration := time.Second * time.Duration(SPLIST_INTERVAL_BASE+rand.Intn(SPLIST_MAX_JITTER))
			utils.DebugLog("scheduled to get sp-list after: ", duration.Seconds(), "second")
			p.ppPeerClock.AddJobWithInterval(duration, p.StartGetSPList(ctx))
		} else {
			retryCounter = 0
		}
	}
}
