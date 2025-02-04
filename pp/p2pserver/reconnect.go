package p2pserver

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/setting"
)

var optimalSpNetworkAddr string
var networkAddMu sync.Mutex

// ConnectToSP checks if there is a connection to an SP node. If there isn't, it attempts to create one with a random SP node.
func (p *P2pServer) ConnectToSP(ctx context.Context) (newConnection bool, err error) {
	if p.SpConnValid() {
		return false, nil
	}
	spList := setting.GetSPList()
	if len(spList) == 0 {
		return false, errors.New("there are no known SP nodes")
	}

	networkAddMu.Lock()
	if optimalSpNetworkAddr != "" {
		_ = p.NewClientToMainSp(ctx, optimalSpNetworkAddr)
		optimalSpNetworkAddr = ""
		networkAddMu.Unlock()
		if p.SpConnValid() {
			pp.DebugLog(ctx, "reconnected to detected optimal SP ", optimalSpNetworkAddr)
			return true, nil
		} else {
			pp.DebugLog(ctx, "can't reconnect to detected optimal SP ", optimalSpNetworkAddr)
		}
	} else {
		networkAddMu.Unlock()
	}

	// Select a random SP node to connect to
	spListOrder := rand.Perm(len(spList))
	for _, index := range spListOrder {
		selectedSP := spList[index]
		if !p.CanConnectToSp(selectedSP.P2PAddress) {
			continue
		}

		_ = p.NewClientToMainSp(ctx, selectedSP.NetworkAddress)
		if p.SpConnValid() {
			pp.DebugLog(ctx, "NewClient:", selectedSP.NetworkAddress)
			return true, nil
		} else {
			pp.DebugLog(ctx, "Can't connect to SP ", selectedSP.NetworkAddress)
			_ = p.RecordSpMaintenance(selectedSP.P2PAddress, time.Now())
		}
	}

	return false, errors.New("couldn't connect to any SP node")
}

// ConfirmOptSP connect if there is a detected optimal SP node.
func (p *P2pServer) ConfirmOptSP(ctx context.Context, spNetworkAddr string) {
	networkAddMu.Lock()
	defer networkAddMu.Unlock()

	spName := p.GetSpName()
	if spName == spNetworkAddr {
		pp.DebugLog(ctx, "optimal SP already in connection, won't change SP")
		return
	}

	pp.DebugLog(ctx, "current sp ", spName, " to be altered to new optimal SP ", spNetworkAddr)
	optimalSpNetworkAddr = spNetworkAddr
	go func() {
		if p.mainSpConn != nil {
			p.mainSpConn.ClientClose(true)
		}
	}()
}
