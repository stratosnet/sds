package p2pserver

import (
	"context"
	"math/rand"
	"sync"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/setting"
)

var optimalSpNetworkAddr string
var networkAddMu sync.Mutex

// ConnectToSP Checks if there is a connection to an SP node. If it doesn't, it attempts to create one with a random SP node.
func (p *P2pServer) ConnectToSP(ctx context.Context) (newConnection bool, err error) {
	if p.mainSpConn != nil {
		return false, nil
	}
	spList := setting.GetSPList()
	if len(spList) == 0 {
		return false, errors.New("there are no known SP nodes")
	}

	networkAddMu.Lock()
	if optimalSpNetworkAddr != "" {
		pp.DebugLog(ctx, "reconnect to detected optimal SP ", optimalSpNetworkAddr)
		_ = p.NewClientToMainSp(ctx, optimalSpNetworkAddr)
		optimalSpNetworkAddr = ""
		networkAddMu.Unlock()
		if p.mainSpConn != nil {
			return true, nil
		}
	} else {
		networkAddMu.Unlock()
	}

	// Select a random SP node to connect to
	spListOrder := rand.Perm(len(spList))
	for _, index := range spListOrder {
		selectedSP := spList[index]
		pp.DebugLog(ctx, "NewClient:", selectedSP.NetworkAddress)
		_ = p.NewClientToMainSp(ctx, selectedSP.NetworkAddress)
		if p.mainSpConn != nil {
			return true, nil
		}
	}

	return false, errors.New("couldn't connect to any SP node")
}

// ConfirmOptSP connect if there is a detected optimal SP node.
func (p *P2pServer) ConfirmOptSP(ctx context.Context, spNetworkAddr string) {
	networkAddMu.Lock()
	defer networkAddMu.Unlock()
	if p.mainSpConn != nil {
		if p.mainSpConn.GetName() == spNetworkAddr {
			pp.DebugLog(ctx, "optimal SP already in connection, won't change SP")
			return
		}
	}
	pp.DebugLog(ctx, "current sp ", p.mainSpConn.GetName(), " to be altered to new optimal SP ", spNetworkAddr)
	optimalSpNetworkAddr = spNetworkAddr
	p.mainSpConn.ClientClose(true)
}
