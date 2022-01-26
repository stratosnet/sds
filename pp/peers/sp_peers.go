package peers

import (
	"context"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/setting"

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
func SendMessage(conn core.WriteCloser, pb proto.Message, cmd string) {
	data, err := proto.Marshal(pb)

	if err != nil {
		utils.ErrorLog("error decoding")
		return
	}
	msg := &msg.RelayMsgBuf{
		MSGHead: requests.PPMsgHeader(data, cmd),
		MSGData: data,
	}
	switch conn.(type) {
	case *core.ServerConn:
		conn.(*core.ServerConn).Write(msg)
	case *cf.ClientConn:
		conn.(*cf.ClientConn).Write(msg)
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
	if netid, ok := RegisterPeerMap.Load(p2pAddress); ok {
		utils.Log("transfer to netid = ", netid)
		GetPPServer().Unicast(netid.(int64), msgBuf)
	} else {
		utils.DebugLog("waller ===== ", p2pAddress)
	}
}

// GetPPList P node get PPList
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
	utils.DebugLog("[SP_LATENCY_CHECK] SendHeartbeat(", server, ", req, header.ReqHeartbeat)")
	var spConn *cf.ClientConn
	if client.SPConn.GetName() != server {
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
		if client.SPConn.GetName() != server {
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
