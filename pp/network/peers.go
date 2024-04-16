package network

import (
	"context"
	"sync"

	"github.com/alex023/clock"

	"github.com/stratosnet/sds/framework/utils"

	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/types"
)

type Network struct {
	ppPeerClock          *clock.Clock
	pingTimeSPMap        *sync.Map
	reloadConnectSpRetry int
	reloadRegisterRetry  int
	reloadConnecting     bool
	fsm                  utils.Fsm
}

func GetPeer(ctx context.Context) *Network {
	if ctx == nil || ctx.Value(types.PP_NETWORK_KEY) == nil {
		panic("Network is not instantiated")
	}

	ps := ctx.Value(types.PP_NETWORK_KEY).(*Network)
	return ps
}

func (p *Network) StartPP(ctx context.Context) {
	p.ppPeerClock = clock.NewClock()
	p.pingTimeSPMap = &sync.Map{}
	p.InitFsm()
	p.StartGetSPList(ctx)()
	p.ScheduleSpLatencyCheck(ctx)
	p.StartStatusReportToSP(ctx)
	go p.ListenOffline(ctx)
}

func (p *Network) ListenOffline(ctx context.Context) {
	var qch chan bool
	if v := ctx.Value(types.LISTEN_OFFLINE_QUIT_CH_KEY); v != nil {
		qch = v.(chan bool)
		utils.DebugLogf("ListenOffline quit ch found")
	}
	p.reloadConnectSpRetry = 0
	p.reloadConnecting = false
	for {
		select {
		case offline := <-p2pserver.GetP2pServer(ctx).ReadOfflineChan():
			if offline.IsSp {
				utils.DebugLogf("SP %v has disconnected", offline.NetworkAddress)
				p.reloadConnectSP(ctx)
			} else {
				utils.DebugLogf("PP %v has disconnected", offline.NetworkAddress)
			}
		case <-qch:
			utils.Log("ListenOffline goroutine terminated")
			return
		}
	}
}

func (p *Network) ClearPingTimeMap() {
	p.pingTimeSPMap = &sync.Map{}
}

func (p *Network) StorePingTimeMap(server string, start int64) {
	p.pingTimeSPMap.Store(server, start)
}

func (p *Network) LoadPingTimeMap(key string) (int64, bool) {
	if start, ok := p.pingTimeSPMap.Load(key); ok {
		return start.(int64), true
	} else {
		return 0, false
	}
}

func (p *Network) DeletePingTimeMap(key string) {
	p.pingTimeSPMap.Delete(key)
}
