package network

import (
	"context"
	"sync"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
)

type Network struct {
	ppPeerClock          *clock.Clock
	pingTimePPMap        *sync.Map
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
	p.pingTimePPMap = &sync.Map{}
	p.pingTimeSPMap = &sync.Map{}
	p.InitFsm()
	p.StartGetSPList(ctx)()
	p.ScheduleSpLatencyCheck(ctx)
	p.SchedulePpLatencyCheck(ctx)
	p.StartStatusReportToSP(ctx)
	go p.ListenOffline(ctx)
}

func (p *Network) InitPeer(ctx context.Context) {
	p.GetSPList(ctx)()
	p.GetPPStatusInitPPList(ctx)
	go p.ListenOffline(ctx)
}

func (p *Network) InitPPList(ctx context.Context) {
	pplist, total, _ := p2pserver.GetP2pServer(ctx).GetPPList(ctx)
	if total == 0 {
		p.GetPPListFromSP(ctx)
	} else {
		if success := p.ConnectToGatewayPP(ctx, pplist); !success {
			p.GetPPListFromSP(ctx)
			return
		}
	}
}

func (p *Network) ConnectToGatewayPP(ctx context.Context, pplist []*types.PeerInfo) bool {
	for _, ppInfo := range pplist {
		if ppInfo.NetworkAddress == setting.NetworkAddress {
			p2pserver.GetP2pServer(ctx).DeletePPByNetworkAddress(ctx, ppInfo.NetworkAddress)
			continue
		}
		ppConn, err := p2pserver.GetP2pServer(ctx).NewClientToPp(ctx, ppInfo.NetworkAddress, true)
		if ppConn != nil {
			p2pserver.GetP2pServer(ctx).SetPpClientConn(ppConn)
			return true
		}
		pp.DebugLogf(ctx, "failed to connect to PP %v: %v", ppInfo, utils.FormatError(err))
		p2pserver.GetP2pServer(ctx).DeletePPByNetworkAddress(ctx, ppInfo.NetworkAddress)
	}
	return false
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
				utils.DebugLogf("SP %v is offline", offline.NetworkAddress)
				p.reloadConnectSP(ctx)
			} else {
				p2pserver.GetP2pServer(ctx).PPDisconnected(ctx, "", offline.NetworkAddress)
				p.InitPPList(ctx)
			}
		case <-qch:
			utils.Log("ListenOffline goroutine terminated")
			return
		}
	}
}

func (p *Network) ClearPingTimeMap(sp bool) {
	if sp {
		p.pingTimeSPMap = &sync.Map{}
	} else {
		p.pingTimePPMap = &sync.Map{}
	}
}

func (p *Network) StorePingTimeMap(server string, start int64, sp bool) {
	if sp {
		p.pingTimeSPMap.Store(server, start)
	} else {
		p.pingTimePPMap.Store(server, start)
	}
}

func (p *Network) LoadPingTimeMap(key string, sp bool) (int64, bool) {
	chosenMap := p.pingTimePPMap
	if sp {
		chosenMap = p.pingTimeSPMap
	}

	if start, ok := chosenMap.Load(key); ok {
		return start.(int64), true
	} else {
		return 0, false
	}
}

func (p *Network) DeletePingTimeMap(key string, sp bool) {
	if sp {
		p.pingTimeSPMap.Delete(key)
	} else {
		p.pingTimePPMap.Delete(key)
	}
}
