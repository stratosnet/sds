package p2pserver

import (
	"context"
	"time"

	"github.com/stratosnet/sds-api/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/types"
)

// GetPPList
func (p *P2pServer) GetPPList(ctx context.Context) (list []*types.PeerInfo, total int64, connected int64) {
	list, total, connected = p.peerList.GetPPList(ctx)
	return
}

// SavePPList will save the target list to local list
func (p *P2pServer) SavePPList(ctx context.Context, target *protos.RspGetPPList) error {
	return p.peerList.SavePPList(ctx, target)
}

// GetPPByP2pAddress
func (p *P2pServer) GetPPByP2pAddress(ctx context.Context, p2pAddr string) *types.PeerInfo {
	return p.peerList.GetPPByP2pAddress(ctx, p2pAddr)
}

// DeletePPByNetworkAddress
func (p *P2pServer) DeletePPByNetworkAddress(ctx context.Context, p2pAddr string) {
	p.peerList.DeletePPByNetworkAddress(ctx, p2pAddr)
}

// UpdatePP will update one pp info to local list
func (p *P2pServer) UpdatePP(ctx context.Context, pp *types.PeerInfo) {
	p.peerList.UpdatePP(ctx, pp)
}

func (p *P2pServer) PPDisconnected(ctx context.Context, p2pAddress, networkAddress string) {
	ppNode := p.GetPPByP2pAddress(ctx, p2pAddress)
	if ppNode == nil {
		ppNode = p.peerList.GetPPByNetworkAddress(ctx, networkAddress)
	}

	if ppNode == nil {
		pp.DebugLogf(ctx, "PP %v (%v) is offline. It was not in the local PP list", p2pAddress, networkAddress)
	} else {
		ppNode.Status = types.PEER_NOT_CONNECTED
		ppNode.LastConnectionTime = time.Now().Unix()
		pp.DebugLogf(ctx, "PP %v is offline", ppNode)

		err := p.peerList.SavePPListToFile(ctx)
		if err != nil {
			pp.ErrorLog(ctx, "Error when saving PP list to file", err)
		}
	}
}

func (p *P2pServer) PPDisconnectedNetId(ctx context.Context, netId int64) {
	found := false
	p.peerList.PpMapByNetworkAddress.Range(func(k, v interface{}) bool {
		ppNode, ok := v.(*types.PeerInfo)
		if !ok {
			pp.ErrorLogf(ctx, "Invalid PP with network address %v in local PP list)", k)
			return true
		}
		if ppNode.Status == types.PEER_CONNECTED && ppNode.NetId == netId {
			p.PPDisconnected(ctx, ppNode.P2pAddress, ppNode.NetworkAddress)
			found = true
			return false
		}
		return true
	})

	if !found {
		pp.DebugLogf(ctx, "PP with netId %v is offline, but it was not found in the local PP list", netId)
	}
}
