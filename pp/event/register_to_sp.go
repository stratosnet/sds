package event

// Author j
import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
)

// ReqRegister if get this, must be PP
func ReqRegister(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqRegister
	if requests.UnmarshalData(ctx, &target) {
		peers.UpdatePP(ctx, &types.PeerInfo{
			NetworkAddress: target.Address.NetworkAddress,
			P2pAddress:     target.Address.P2PAddress,
			RestAddress:    target.Address.RestAddress,
			WalletAddress:  target.Address.WalletAddress,
			NetId:          core.NetIDFromContext(ctx),
			Status:         types.PEER_CONNECTED,
		})
		peers.TransferSendMessageToSPServer(ctx, requests.ReqRegisterDataTR(&target))

		// IPProt := strings.Split(target.Address.NetworkAddress, ":")
		// ip := ""
		// port := ""
		// if len(IPProt) > 1 {
		// 	ip = IPProt[0]
		// 	port = IPProt[1]
		// }
		// if ip == "127.0.0.1" {
		//
		// 	utils.DebugLog("user didn't config network address")
		// 	utils.DebugLog("target", target)
		// 	req := target
		// 	req.Address = &protos.PPBaseInfo{
		// 		WalletAddress:  target.Address.WalletAddress,
		// 		NetworkAddress: conn.(*core.ServerConn).GetIP() + ":" + port,
		// 	}
		// 	utils.DebugLog("req", req)
		// 	SendMessageToSPServer(&req, header.ReqRegister)
		// } else {
		// 	// transfer to SP
		// 	transferSendMessageToSPServer(reqRegisterDataTR(&target))
		// }
	}
}

// RspRegister  PP -> SP, SP -> PP, PP -> P
func RspRegister(ctx context.Context, conn core.WriteCloser) {
	pp.Log(ctx, "get RspRegister", conn)
	var target protos.RspRegister
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	pp.Log(ctx, "target.RspRegister", target.P2PAddress)
	if target.P2PAddress != setting.P2PAddress {
		pp.Log(ctx, "transfer RspRegister to: ", target.P2PAddress)
		peers.TransferSendMessageToPPServByP2pAddress(ctx, target.P2PAddress, core.MessageFromContext(ctx))
		return
	}

	pp.Log(ctx, "get RspRegister ", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		//setting.P2PAddress = ""
		//setting.WalletAddress = ""
		pp.Log(ctx, "Register failed", target.Result.Msg)
		return
	}

	pp.Log(ctx, "Register successful", target.Result.Msg)
	setting.IsLoad = true
	setting.IsLoginToSP = true
	pp.DebugLog(ctx, "@@@@@@@@@@@@@@@@@@@@@@@@@@@@", client.GetConnectionName(conn))
	setting.IsPP = target.IsPP
	if !setting.IsPP {
		reportDHInfoToPP(ctx)
	}
	if setting.IsPP && !setting.IsStartMining {
		peers.StartMining(ctx)
	}
}

// RspMining RspMining
func RspMining(ctx context.Context, conn core.WriteCloser) {
	pp.DebugLog(ctx, "get RspMining", conn)
	var target protos.RspMining
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	if target.Result.State != protos.ResultState_RES_SUCCESS {
		pp.Log(ctx, target.Result.Msg)
		return
	}

	pp.Log(ctx, "start mining")
	if peers.GetPPServer() == nil {
		go peers.StartListenServer(ctx, setting.Config.Port)
	}
	setting.IsStartMining = true

	newConnection, err := peers.ConnectToSP(ctx)
	if err != nil {
		utils.ErrorLog(err)
		return
	}
	if newConnection {
		peers.RegisterToSP(ctx, true)
	}

	pp.DebugLog(ctx, "Start reporting node status to SP")
	// trigger 1 stat report immediately
	peers.ReportNodeStatus(ctx)()
}
