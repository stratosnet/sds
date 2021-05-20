package event

// Author j
import (
	"context"
	"fmt"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/spbf"
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
		SendMessageToSPServer(reqRegisterData(toSP), header.ReqRegister)
		utils.Log("SendMessage(conn, req, header.ReqRegister) to SP")
	} else {
		sendMessage(client.PPConn, reqRegisterData(toSP), header.ReqRegister)
		utils.Log("SendMessage(conn, req, header.ReqRegister) to PP")
	}

}

// ReqRegisterChain if get this, must be PP
func ReqRegisterChain(ctx context.Context, conn spbf.WriteCloser) {
	utils.Log("PP get ReqRegisterChain")
	var target protos.ReqRegister
	if unmarshalData(ctx, &target) {
		// store register P wallet address
		serv.RegisterPeerMap.Store(target.Address.WalletAddress, spbf.NetIDFromContext(ctx))
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
		// 		NetworkAddress: conn.(*spbf.ServerConn).GetIP() + ":" + port,
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
func RspRegisterChain(ctx context.Context, conn spbf.WriteCloser) {
	utils.Log("get RspRegisterChain", conn)
	var target protos.RspRegister
	if !unmarshalData(ctx, &target) {
		return
	}
	utils.Log("target.RspRegister", target.WalletAddress)
	if target.WalletAddress != setting.WalletAddress {
		utils.Log("transfer RspRegisterChain to: ", target.WalletAddress)
		transferSendMessageToClient(target.WalletAddress, spbf.MessageFromContext(ctx))
		return
	}

	utils.Log("get RspRegisterChain ", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
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
}
