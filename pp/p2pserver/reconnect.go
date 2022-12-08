package p2pserver

import (
	"context"
	"math/rand"
	"sync"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/setting"
)

type OptimalSp struct {
	networkAddr string
	mtx         sync.Mutex
}

var (
	optSp               = &OptimalSp{}
	minReloadSpInterval = 3
	maxReloadSpInterval = 900 //15 min
	retry               = 0
)

// ConnectToSP Checks if there is a connection to an SP node. If it doesn't, it attempts to create one with a random SP node.
func (p *P2pServer) ConnectToSP(ctx context.Context) (newConnection bool, err error) {
	if p.mainSpConn != nil {
		return false, nil
	}
	if len(setting.Config.SPList) == 0 {
		return false, errors.New("there are no SP nodes in the config file")
	}

	if optSpNetworkAddr, err := p.GetOptSPAndClear(); err == nil {
		pp.DebugLog(ctx, "reconnect to detected optimal SP ", optSpNetworkAddr)
		_ = p.NewClientToMainSp(ctx, optSpNetworkAddr)
		if p.mainSpConn != nil {
			return true, nil
		}
	}
	// Select a random SP node to connect to
	spListOrder := rand.Perm(len(setting.Config.SPList))
	for _, index := range spListOrder {
		selectedSP := setting.Config.SPList[index]
		pp.DebugLog(ctx, "NewClient:", selectedSP.NetworkAddress)
		_ = p.NewClientToMainSp(ctx, selectedSP.NetworkAddress)
		if p.mainSpConn != nil {
			return true, nil
		}
	}

	return false, errors.New("couldn't connect to any SP node")
}

// ConnectToOptSP connect if there is a detected optimal SP node.
func (p *P2pServer) ConfirmOptSP(ctx context.Context, spNetworkAddr string) {
	pp.DebugLog(ctx, "current sp ", p.mainSpConn.GetName(), " to be altered to new optimal SP ", spNetworkAddr)
	if p.mainSpConn.GetName() == spNetworkAddr {
		pp.DebugLog(ctx, "optimal SP already in connection, won't change SP")
		return
	}
	p.setOptSP(spNetworkAddr)
	p.mainSpConn.ClientClose(true)
}

func (p *P2pServer) GetOptSPAndClear() (string, error) {
	if len(optSp.networkAddr) > 0 {
		optSpNetworkAddr := optSp.networkAddr
		optSp = &OptimalSp{}
		return optSpNetworkAddr, nil
	}
	return "", errors.New("optimal SP not detected")
}

func (p *P2pServer) setOptSP(spNetworkAddr string) {
	optSp.networkAddr = spNetworkAddr
}
