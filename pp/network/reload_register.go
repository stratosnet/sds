package network

import (
	"context"
	"math/rand"
	"time"

	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
)

const (
	MIN_RELOAD_REGISTER_INTERVAL = 60
	MAX_RELOAD_REGISTER_INTERVAL = 600
)

// StartStatusReportToSP to start a timer scheduling reporting Node Status to SP
func (p *Network) StartRegisterToSp(ctx context.Context) {
	p.tryRegister(ctx)()
	p.reloadConnectSpRetry = 0
}

func (p *Network) tryRegister(ctx context.Context) func() {
	return func() {
		if !setting.IsLoginToSP && setting.State == types.PP_ACTIVE && !setting.IsSuspended {
			utils.DebugLog("Send register and set next try")
			p.RegisterToSP(ctx, true)

			p.reloadRegisterRetry += 1
			reloadInterval := MIN_RELOAD_REGISTER_INTERVAL*p.reloadRegisterRetry + rand.Intn(MIN_RELOAD_REGISTER_INTERVAL)
			//prevent reloadSpInterval from overflowing after multiple reloadConnectSpRetry
			if reloadInterval > MAX_RELOAD_REGISTER_INTERVAL {
				reloadInterval = MAX_RELOAD_REGISTER_INTERVAL
			}

			p.ppPeerClock.AddJobWithInterval(time.Second*time.Duration(reloadInterval), p.tryRegister(ctx))
		} else {
			utils.DebugLog("Register process done, no more retry")
		}
	}
}
