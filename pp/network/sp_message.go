package network

import (
	"context"

	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
)

// RegisterToSP send ReqRegister to SP
func (p *Network) RegisterToSP(ctx context.Context, toSP bool) {
	if toSP {
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqRegisterData(), header.ReqRegister)
		pp.Log(ctx, "SendMessage(conn, req, header.ReqRegister) to SP")
	} else {
		p2pserver.GetP2pServer(ctx).SendMessage(ctx, p2pserver.GetP2pServer(ctx).GetPpConn(), requests.ReqRegisterData(), header.ReqRegister)
		pp.Log(ctx, "SendMessage(conn, req, header.ReqRegister) to PP")
	}
}

// StartMining send ReqMining to SP if needed
func (p *Network) StartMining(ctx context.Context) {
	if setting.CheckLogin() {
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqMiningData(), header.ReqMining)
	}
}

// GetPPStatusFromSP send ReqGetPPStatus to SP
func (p *Network) GetPPStatusFromSP(ctx context.Context) {
	pp.DebugLog(ctx, "SendMessage(client.spConn, req, header.ReqGetPPStatus)")
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqGetPPStatusData(false), header.ReqGetPPStatus)
}

// GetPPStatusInitPPList P node get node status
func (p *Network) GetPPStatusInitPPList(ctx context.Context) func() {
	return func() {
		pp.DebugLogf(ctx, "SendMessage(client.spConn, req, header.ReqGetPPStatus)")
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqGetPPStatusData(true), header.ReqGetPPStatus)
	}
}

// GetSPList node get spList
func (p *Network) GetSPList(ctx context.Context) func() {
	return func() {
		pp.DebugLogf(ctx, "SendMessage(client.spConn, req, header.ReqGetSPList)")
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqGetSPlistData(), header.ReqGetSPList)
	}
}

// GetPPListFromSP node get ppList from sp
func (p *Network) GetPPListFromSP(ctx context.Context) {
	pp.DebugLogf(ctx, "SendMessage(client.spConn, req, header.ReqGetPPList)")
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqGetPPlistData(), header.ReqGetPPList)
}
