package network

import (
	"context"
	"math"
	"time"

	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/setting"
)

var (
	minReloadSpInterval = 3
	maxReloadSpInterval = 900 //15 min
	retry               = 0
)

// reloadConnectSP
func (p *Network) reloadConnectSP(ctx context.Context) func() {
	return func() {
		newConnection, err := p2pserver.GetP2pServer(ctx).ConnectToSP(ctx)
		if newConnection {
			p.RegisterToSP(ctx, true)
			retry = 0
			if setting.IsStartMining {
				p.StartMining(ctx)
			}
		}

		if err != nil {
			//calc next reload interval
			reloadSpInterval := minReloadSpInterval * int(math.Ceil(math.Pow(10, float64(retry)))) * 2
			//prevent reloadSpInterval from overflowing after multiple retry
			if reloadSpInterval < maxReloadSpInterval {
				retry += 1
			}
			reloadSpInterval = int(math.Min(float64(reloadSpInterval), float64(maxReloadSpInterval)))
			pp.Logf(ctx, "couldn't connect to SP node. Retrying in %v seconds...", reloadSpInterval)
			p.ppPeerClock.AddJobWithInterval(time.Duration(reloadSpInterval)*time.Second, p.reloadConnectSP(ctx))
		}
	}
}
