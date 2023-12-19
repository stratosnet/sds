package network

import (
	"context"
	"time"

	"github.com/stratosnet/sds/framework/msg/header"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
)

// RegisterToSP send ReqRegister to SP
func (p *Network) RegisterToSP(ctx context.Context, toSP bool) {
	nowSec := time.Now().Unix()
	//// sign the wallet signature by wallet private key
	wsignMsg := utils.RegisterWalletSignMessage(setting.WalletAddress, nowSec)
	wsign, err := setting.WalletPrivateKey.Sign([]byte(wsignMsg))
	if err != nil {
		return
	}
	if toSP {
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx,
			requests.ReqRegisterData(ctx, setting.WalletAddress, setting.WalletPublicKey.Bytes(), wsign, nowSec),
			header.ReqRegister)
		pp.Log(ctx, "SendMessage(conn, req, header.ReqRegister) to SP")
	} else {
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, p2pserver.GetP2pServer(ctx).GetPpConn(),
			requests.ReqRegisterData(ctx, setting.WalletAddress, setting.WalletPublicKey.Bytes(), wsign, nowSec),
			header.ReqRegister)
		pp.Log(ctx, "SendMessage(conn, req, header.ReqRegister) to PP")
	}
}

// StartMining send ReqMining to SP if needed
func (p *Network) StartMining(ctx context.Context) {
	if setting.CheckLogin() {
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqMiningData(ctx), header.ReqMining)
	}
}

// GetPPStatusFromSP send ReqGetPPStatus to SP
func (p *Network) GetPPStatusFromSP(ctx context.Context) {
	pp.DebugLog(ctx, "SendMessage(client.spConn, req, header.ReqGetPPStatus)")
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqGetPPStatusData(ctx, false), header.ReqGetPPStatus)
}

// GetPPStatusInitPPList P node get node status
func (p *Network) GetPPStatusInitPPList(ctx context.Context) {
	pp.DebugLogf(ctx, "SendMessage(client.spConn, req, header.ReqGetPPStatus)")
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqGetPPStatusData(ctx, true), header.ReqGetPPStatus)
}

// GetSPList node get spList
func (p *Network) GetSPList(ctx context.Context) func() {
	return func() {
		pp.DebugLogf(ctx, "SendMessage(client.spConn, req, header.ReqGetSPList)")
		nowSec := time.Now().Unix()
		wsignMsg := utils.GetSPListWalletSignMessage(setting.WalletAddress, nowSec)
		wsign, err := setting.WalletPrivateKey.Sign([]byte(wsignMsg))
		if err != nil {
			return
		}
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx,
			requests.ReqGetSPlistData(ctx, setting.WalletAddress, setting.WalletPublicKey.Bytes(), wsign, nowSec),
			header.ReqGetSPList)
	}
}

// GetPPListFromSP node get ppList from sp
func (p *Network) GetPPListFromSP(ctx context.Context) {
	pp.DebugLogf(ctx, "SendMessage(client.spConn, req, header.ReqGetPPList)")
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqGetPPlistData(ctx), header.ReqGetPPList)
}
