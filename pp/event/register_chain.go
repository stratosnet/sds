package event

// Author j
import (
	"context"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
)

// ReqRegister if get this, must be PP
func ReqRegister(ctx context.Context, conn core.WriteCloser) {
	utils.Log("PP get ReqRegister")
	var target protos.ReqRegister
	if types.UnmarshalData(ctx, &target) {
		// store register P wallet address
		peers.RegisterPeerMap.Store(target.Address.P2PAddress, core.NetIDFromContext(ctx))
		peers.TransferSendMessageToSPServer(types.ReqRegisterDataTR(&target))

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
	utils.Log("get RspRegister", conn)
	var target protos.RspRegister
	if !types.UnmarshalData(ctx, &target) {
		return
	}
	utils.Log("target.RspRegister", target.P2PAddress)
	if target.P2PAddress != setting.P2PAddress {
		utils.Log("transfer RspRegister to: ", target.P2PAddress)
		peers.TransferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		return
	}

	utils.Log("get RspRegister ", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		setting.P2PAddress = ""
		setting.WalletAddress = ""
		utils.Log("login failed", target.Result.Msg)
		return
	}

	utils.Log("login successful", target.Result.Msg)
	setting.IsLoad = true
	utils.DebugLog("@@@@@@@@@@@@@@@@@@@@@@@@@@@@", conn.(*cf.ClientConn).GetName())
	setting.IsPP = target.IsPP
	if !setting.IsPP {
		reportDHInfoToPP()
	}
	if setting.IsAuto {
		if setting.IsPP {
			peers.StartMining()
		}
	}

}

// RspMining RspMining
func RspMining(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get RspMining", conn)
	var target protos.RspMining
	if types.UnmarshalData(ctx, &target) {
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			utils.Log("start mining")
			if peers.GetPPServer() == nil {
				go peers.StartListenServer(setting.Config.Port)
			}
			setting.IsStartMining = true

			newConnection, err := peers.ConnectToSP()
			if err != nil {
				utils.ErrorLog(err)
				return
			}
			if newConnection {
				peers.RegisterChain(true)
			}

			utils.DebugLog("Start reporting node status to SP")
		} else {
			utils.Log(target.Result.Msg)
		}
	}
}
