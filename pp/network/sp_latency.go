package network

import (
	"context"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/sds-msg/header"
	"github.com/stratosnet/sds/sds-msg/protos"
)

type CandidateSp struct {
	NetworkAddr        string
	SpResponseTimeCost int64
}

var (
	candidateSps []CandidateSp
	mtx          sync.Mutex
)

func (p *Network) GetSpCandidateList() []CandidateSp {
	return candidateSps
}

func (p *Network) ClearSpCandidateList() {
	candidateSps = nil
}

func (p *Network) UpdateSpCandidateList(c CandidateSp) {
	mtx.Lock()
	defer mtx.Unlock()
	for i, candidate := range candidateSps {
		if candidate.NetworkAddr == c.NetworkAddr {
			candidateSps[i].SpResponseTimeCost = c.SpResponseTimeCost
			return
		}
	}
	candidateSps = append(candidateSps, c)
}

func (p *Network) ScheduleSpLatencyCheck(ctx context.Context) {
	p.ppPeerClock.AddJobRepeat(time.Second*utils.LatencyCheckSpListInterval, 0, p.SpLatencyCheck(ctx))
}

func (p *Network) SpLatencyCheck(ctx context.Context) func() {
	return func() {
		mtx.Lock()
		defer mtx.Unlock()

		if !p2pserver.GetP2pServer(ctx).SpConnValid() {
			utils.DebugLog("SP latency check skipped until connection to SP is recovered")
			return
		}

		setting.SPMap.Range(func(k, v any) bool {
			selectedSP := v.(setting.SPBaseInfo)
			server := selectedSP.NetworkAddress
			utils.DebugLog("[SP_LATENCY_CHECK] SendSpLatencyCheck(", server, ", req, header.ReqSpLatencyCheck)")
			var spConn *cf.ClientConn
			var err error
			if p2pserver.GetP2pServer(ctx).GetSpName() != server {
				spConn, err = p2pserver.GetP2pServer(ctx).NewClientToAlternativeSp(ctx, server)
				if err != nil {
					utils.DebugLogf("failed to connect to server %v: %v", server, utils.FormatError(err))
				}
			} else {
				utils.DebugLog("Checking latency for working SP ", server)
				spConn = p2pserver.GetP2pServer(ctx).GetSpConn()
			}
			if spConn != nil {
				start := time.Now().UnixNano()
				p.StorePingTimeMap(server, start, true)
				pb := &protos.ReqSpLatencyCheck{
					P2PAddressPp:     p2pserver.GetP2pServer(ctx).GetP2PAddress(),
					NetworkAddressSp: server,
				}
				_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, spConn, pb, header.ReqSpLatencyCheck)
				if p2pserver.GetP2pServer(ctx).GetSpName() != server {
					p2pserver.GetP2pServer(ctx).StoreBufferedSpConn(spConn)
				}
			}
			return true
		})
		p.ppPeerClock.AddJobRepeat(time.Second*utils.LatencyCheckSpListTimeout, 1, p.ChooseSpToConnectTo(ctx))
	}
}

func (p *Network) ChooseSpToConnectTo(ctx context.Context) func() {
	return func() {
		mtx.Lock()
		defer mtx.Unlock()
		// clear buffered spConn
		spConnsToClose := p2pserver.GetP2pServer(ctx).GetBufferedSpConns()
		pp.DebugLog(ctx, "ChooseSpToConnectTo")
		pp.DebugLogf(ctx, "There are %v spConn", len(spConnsToClose))
		for _, spConn := range spConnsToClose {
			if p2pserver.GetP2pServer(ctx).SpConnValid() && spConn.GetName() == p2pserver.GetP2pServer(ctx).GetSpName() {
				pp.DebugLogf(ctx, "spConn %v is the current main connection, not closing it", spConn.GetName())
			} else {
				pp.DebugLogf(ctx, "closing spConn %v", spConn.GetName())
				spConn.Close()
			}
		}
		if len(candidateSps) == 0 {
			pp.ErrorLog(ctx, "No candidate optimal SP")
			return
		}
		sort.Slice(candidateSps, func(i, j int) bool {
			return candidateSps[i].SpResponseTimeCost < candidateSps[j].SpResponseTimeCost
		})
		nSpsConsidered := utils.LatencyCheckTopSpsConsidered // Select from top 3 SPs
		if nSpsConsidered > len(candidateSps) {
			nSpsConsidered = len(candidateSps)
		}

		selectedSp := rand.Intn(nSpsConsidered)
		p2pserver.GetP2pServer(ctx).ConfirmOptSP(ctx, candidateSps[selectedSp].NetworkAddr)
	}
}
