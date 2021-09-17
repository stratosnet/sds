package event

// Author j
import (
	"context"
	"fmt"
	"github.com/alex023/clock"
	"time"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/serv"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// RegisterChain
func RegisterChain(toSP bool) {
	if toSP {
		SendMessageToSPServer(reqRegisterData(), header.ReqRegister)
		utils.Log("SendMessage(conn, req, header.ReqRegister) to SP")
	} else {
		sendMessage(client.PPConn, reqRegisterData(), header.ReqRegister)
		utils.Log("SendMessage(conn, req, header.ReqRegister) to PP")
	}

}

// ReqRegisterChain if get this, must be PP
func ReqRegisterChain(ctx context.Context, conn core.WriteCloser) {
	utils.Log("PP get ReqRegisterChain")
	var target protos.ReqRegister
	if unmarshalData(ctx, &target) {
		// store register P wallet address
		serv.RegisterPeerMap.Store(target.Address.P2PAddress, core.NetIDFromContext(ctx))
		transferSendMessageToSPServer(reqRegisterDataTR(&target))

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

// RspRegisterChain  PP -> SP, SP -> PP, PP -> P
func RspRegisterChain(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspRegisterChain", conn)
	var target protos.RspRegister
	if !unmarshalData(ctx, &target) {
		return
	}
	utils.Log("target.RspRegister", target.P2PAddress)
	if target.P2PAddress != setting.P2PAddress {
		utils.Log("transfer RspRegisterChain to: ", target.P2PAddress)
		transferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		return
	}

	utils.Log("get RspRegisterChain ", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		setting.P2PAddress = ""
		setting.WalletAddress = ""
		fmt.Println("login failed", target.Result.Msg)
		return
	}

	fmt.Println("login successfully", target.Result.Msg)
	setting.IsLoad = true
	utils.DebugLog("@@@@@@@@@@@@@@@@@@@@@@@@@@@@", conn.(*cf.ClientConn).GetName())
	setting.IsPP = target.IsPP
	if !setting.IsPP {
		reportDHInfoToPP()
	}
	if setting.IsAuto {
		if setting.IsPP {
			StartMining()
		}
	}

}

// RspMining RspMining
func RspMining(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get RspMining", conn)
	var target protos.RspMining
	if unmarshalData(ctx, &target) {
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			fmt.Println("start mining")
			if serv.GetPPServer() == nil {
				go serv.StartListenServer(setting.Config.Port)
			}
			setting.IsStartMining = true
			if client.SPConn == nil {
				client.SPConn = client.NewClient(setting.Config.SPNetAddress, setting.IsPP)
				RegisterChain(true)
			}
			utils.DebugLog("Start reporting node status to SP")
			clock := clock.NewClock()
			clock.AddJobRepeat(time.Second*60, 0, ReportNodeStatus)
		} else {
			utils.Log(target.Result.Msg)
		}
	}
}

// StartMining
func StartMining() {
	if setting.CheckLogin() {
		if setting.IsPP {
			utils.DebugLog("StartMining")
			SendMessageToSPServer(reqMiningData(), header.ReqMining)
		} else {
			fmt.Println("register as miner first")
		}
	}
}
