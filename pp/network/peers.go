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
	reloadConnecting bool
	fsm              utils.Fsm
}

const (
	PP_NETWORK_KEY = "PpNetworkKey"
)

// GetPeer
func GetPeer(ctx context.Context) *Network {
	if ctx == nil || ctx.Value(PP_NETWORK_KEY) == nil {
		panic("Network is not instantiated")
	}

	ps := ctx.Value(PP_NETWORK_KEY).(*Network)
	return ps
}

// StartPP
func (p *Network) StartPP(ctx context.Context) {
	p.ppPeerClock = clock.NewClock()
	p.pingTimePPMap = &sync.Map{}
	p.pingTimeSPMap = &sync.Map{}
	p.InitFsm()
	p.GetSPList(ctx)()
	p.GetPPStatusInitPPList(ctx)()
	p.StartPpLatencyCheck(ctx)
	p.StartStatusReportToSP(ctx)
	go p.ListenOffline(ctx)
}

// InitPeer
func (p *Network) InitPeer(ctx context.Context) {
	p.GetSPList(ctx)()
	p.GetPPStatusInitPPList(ctx)()
	go p.ListenOffline(ctx)
}

// InitPPList
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

// ConnectToGatewayPP
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

// ListenOffline
func (p *Network) ListenOffline(ctx context.Context) {
	var qch chan bool
	if v := ctx.Value(p2pserver.LISTEN_OFFLINE_QUIT_CH_KEY); v != nil {
		qch = v.(chan bool)
		utils.DebugLogf("ListenOffline quit ch found")
	}
	p.reloadConnectSpRetry = 0
	p.reloadConnecting = false
	for {
		select {
		case offline := <-p2pserver.GetP2pServer(ctx).ReadOfflineChan():
			if offline.IsSp {
				if setting.IsPP {
					utils.DebugLogf("SP %v is offline", offline.NetworkAddress)
					p.reloadConnectSP(ctx)
				}
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

func (p *Network) ClearPingTimeSPMap() {
	p.pingTimeSPMap = &sync.Map{}
}

func (p *Network) StorePingTimeSPMap(server string, start int64) {
	p.pingTimePPMap.Store(server, start)
}

func (p *Network) LoadPingTimeSPMap(key string) (int64, bool) {
	if start, ok := p.pingTimePPMap.Load(key); ok {
		return start.(int64), true
	} else {
		return 0, false
	}
}

func (p *Network) DeletePingTimePPMap(key string) {
	p.pingTimePPMap.Delete(key)
}
