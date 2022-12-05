package network

import (
	"context"
	"math"
	"time"

	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/setting"
)

const (
	MIN_RELOAD_SP_INTERVAL = 3
	MAX_RELOAD_SP_INTERVAL = 900 //15 min
)

// reloadConnectSP
func (p *Network) reloadConnectSP(ctx context.Context) {
	if p.reloadConnecting {
		return
	}
	p.reloadConnecting = true
	p.tryReloadConnectSP(ctx)()
	p.GetSPList(ctx)()
}

// tryReloadConnectSP
func (p *Network) tryReloadConnectSP(ctx context.Context) func() {
	return func() {
		newConnection, err := p2pserver.GetP2pServer(ctx).ConnectToSP(ctx)
		if newConnection {
			p.RegisterToSP(ctx, true)
			p.reloadConnecting = false
			p.reloadConnectSpRetry = 0
			if setting.IsStartMining {
				p.StartMining(ctx)
			}
		}

		if err != nil {
			//calc next reload interval
			reloadSpInterval := MIN_RELOAD_SP_INTERVAL * int(math.Ceil(math.Pow(10, float64(p.reloadConnectSpRetry)))) * 2
			//prevent reloadSpInterval from overflowing after multiple reloadConnectSpRetry
			if reloadSpInterval < MAX_RELOAD_SP_INTERVAL {
				p.reloadConnectSpRetry += 1
			}
			reloadSpInterval = int(math.Min(float64(reloadSpInterval), float64(MAX_RELOAD_SP_INTERVAL)))
			pp.Logf(ctx, "couldn't connect to SP node. Retrying in %v seconds...", reloadSpInterval)
			p.ppPeerClock.AddJobWithInterval(time.Duration(reloadSpInterval)*time.Second, p.tryReloadConnectSP(ctx))
		}
	}
}
