package peers

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/utils"
)

var bufferedSpConns = make([]*cf.ClientConn, 0)

// SendMessage
func SendMessage(conn core.WriteCloser, pb proto.Message, cmd string) error {
	data, err := proto.Marshal(pb)

	if err != nil {
		utils.ErrorLog("error decoding")
		return errors.New("error decoding")
	}
	msg := &msg.RelayMsgBuf{
		MSGHead: requests.PPMsgHeader(data, cmd),
		MSGData: data,
	}
	switch conn.(type) {
	case *core.ServerConn:
		return conn.(*core.ServerConn).Write(msg)
	case *cf.ClientConn:
		return conn.(*cf.ClientConn).Write(msg)
	default:
		return errors.New("unknown connection type")
	}
}

func SendMessageDirectToSPOrViaPP(pb proto.Message, cmd string) {
	if client.SPConn != nil {
		SendMessage(client.SPConn, pb, cmd)
	} else {
		SendMessage(client.PPConn, pb, cmd)
	}
}

// SendMessageToSPServer SendMessageToSPServer
func SendMessageToSPServer(pb proto.Message, cmd string) {
	_, err := ConnectToSP()
	if err != nil {
		utils.ErrorLog(err)
		return
	}

	SendMessage(client.SPConn, pb, cmd)
}

// TransferSendMessageToPPServ
func TransferSendMessageToPPServ(addr string, msgBuf *msg.RelayMsgBuf) {
	if client.ConnMap[addr] != nil {

		client.ConnMap[addr].Write(msgBuf)
		utils.DebugLog("conn exist, transfer")
	} else {
		utils.DebugLog("new conn, connect and transfer")
		client.NewClient(addr, false).Write(msgBuf)
	}
}

func TransferSendMessageToPPServByP2pAddress(p2pAddress string, msgBuf *msg.RelayMsgBuf) {
	ppInfo := peerList.GetPPByP2pAddress(p2pAddress)
	if ppInfo == nil {
		utils.ErrorLogf("PP %v missing from local ppList. Cannot transfer message due to missing network address", p2pAddress)
		return
	}
	TransferSendMessageToPPServ(ppInfo.NetworkAddress, msgBuf)
}

// transferSendMessageToSPServer
func TransferSendMessageToSPServer(msg *msg.RelayMsgBuf) {
	_, err := ConnectToSP()
	if err != nil {
		utils.ErrorLog(err)
		return
	}

	client.SPConn.Write(msg)
}

// ReqTransferSendSP
func ReqTransferSendSP(ctx context.Context, conn core.WriteCloser) {
	TransferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// transferSendMessageToClient
func TransferSendMessageToClient(p2pAddress string, msgBuf *msg.RelayMsgBuf) {
	pp := peerList.GetPPByP2pAddress(p2pAddress)
	if pp != nil && pp.Status == types.PEER_CONNECTED {
		utils.Log("transfer to netid = ", pp.NetId)
		GetPPServer().Unicast(pp.NetId, msgBuf)
	} else {
		utils.DebugLog("waller ===== ", p2pAddress)
	}
}

// GetMyNodeStatusFromSP P node get node status
func GetPPStatusFromSP() {
	utils.DebugLog("SendMessage(client.SPConn, req, header.ReqGetPPStatus)")
	SendMessageToSPServer(requests.ReqGetPPStatusData(), header.ReqGetPPStatus)
}

// GetSPList node get spList
func GetSPList() {
	utils.DebugLog("SendMessage(client.SPConn, req, header.ReqGetSPList)")
	SendMessageToSPServer(requests.ReqGetSPlistData(), header.ReqGetSPList)
}

func SendLatencyCheckMessageToSPList() {
	utils.DebugLogf("[SP_LATENCY_CHECK] SendHeartbeatToSPList, num of SPs: %v", len(setting.Config.SPList))
	if len(setting.Config.SPList) < 2 {
		utils.ErrorLog("there are not enough SP nodes in the config file")
		return
	}
	for i := 0; i < len(setting.Config.SPList); i++ {
		selectedSP := setting.Config.SPList[i]
		checkSingleSpLatency(selectedSP.NetworkAddress, false)
	}
}

func checkSingleSpLatency(server string, heartbeat bool) {
	if client.SPConn == nil {
		utils.DebugLog("SP latency check skipped until connection to SP is recovered")
		return
	}
	utils.DebugLog("[SP_LATENCY_CHECK] SendHeartbeat(", server, ", req, header.ReqHeartbeat)")
	var spConn *cf.ClientConn
	if client.GetConnectionName(client.SPConn) != server {
		spConn = client.NewClient(server, heartbeat)
	} else {
		utils.DebugLog("Checking latency for working SP ", server)
		spConn = client.SPConn
	}
	//defer spConn.Close()
	if spConn != nil {
		start := time.Now().UnixNano()
		pb := &protos.ReqHeartbeat{
			HbType:           protos.HeartbeatType_LATENCY_CHECK,
			P2PAddressPp:     setting.P2PAddress,
			NetworkAddressSp: server,
			PingTime:         strconv.FormatInt(start, 10),
		}
		SendMessage(spConn, pb, header.ReqHeart)
		if client.GetConnectionName(client.SPConn) != server {
			bufferedSpConns = append(bufferedSpConns, spConn)
		}
	}
}

func GetBufferedSpConns() []*cf.ClientConn {
	return bufferedSpConns
}

func ClearBufferedSpConns() {
	bufferedSpConns = make([]*cf.ClientConn, 0)
}

func ScheduleReloadSPlist(future time.Duration) {
	utils.DebugLog("scheduled to get sp-list after: ", future.Seconds(), "second")
	ppPeerClock.AddJobWithInterval(future, GetSPList)
}

func ScheduleReloadPPStatus(future time.Duration) {
	utils.DebugLog("scheduled to get pp status from sp after: ", future.Seconds(), "second")
	ppPeerClock.AddJobWithInterval(future, GetPPStatusFromSP)
}
