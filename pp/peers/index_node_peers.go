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

var bufferedIndexNodeConns = make([]*cf.ClientConn, 0)

// SendMessage
func SendMessage(conn core.WriteCloser, pb proto.Message, cmd string) error {
	return SendResponseMessageWithReqId(conn, pb, cmd, int64(0))
}

func SendResponseMessageWithReqId(conn core.WriteCloser, pb proto.Message, cmd string, reqId int64) error {
	data, err := proto.Marshal(pb)

	if err != nil {
		utils.ErrorLog("error decoding")
		return errors.New("error decoding")
	}
	msg := &msg.RelayMsgBuf{
		MSGHead: header.MakeMessageHeader(1, uint16(setting.Config.Version.AppVer), uint32(len(data)), cmd, reqId),
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

func SendMessageDirectToIndexNodeOrViaPP(pb proto.Message, cmd string) {
	if client.IndexNodeConn != nil {
		SendMessage(client.IndexNodeConn, pb, cmd)
	} else {
		SendMessage(client.PPConn, pb, cmd)
	}
}

// SendMessageToIndexNodeServer SendMessageToIndexNodeServer
func SendMessageToIndexNodeServer(pb proto.Message, cmd string) {
	_, err := ConnectToIndexNode()
	if err != nil {
		utils.ErrorLog(err)
		return
	}

	SendMessage(client.IndexNodeConn, pb, cmd)
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

// transferSendMessageToIndexNodeServer
func TransferSendMessageToIndexNodeServer(msg *msg.RelayMsgBuf) {
	_, err := ConnectToIndexNode()
	if err != nil {
		utils.ErrorLog(err)
		return
	}

	client.IndexNodeConn.Write(msg)
}

// ReqTransferSendIndexNode
func ReqTransferSendIndexNode(ctx context.Context, conn core.WriteCloser) {
	TransferSendMessageToIndexNodeServer(core.MessageFromContext(ctx))
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

// GetMyNodeStatusFromIndexNode P node get node status
func GetPPStatusFromIndexNode() {
	utils.DebugLog("SendMessage(client.IndexNodeConn, req, header.ReqGetPPStatus)")
	SendMessageToIndexNodeServer(requests.ReqGetPPStatusData(false), header.ReqGetPPStatus)
}

// GetMyNodeStatusFromIndexNode P node get node status
func GetPPStatusInitPPList() {
	utils.DebugLog("SendMessage(client.IndexNodeConn, req, header.ReqGetPPStatus)")
	SendMessageToIndexNodeServer(requests.ReqGetPPStatusData(true), header.ReqGetPPStatus)
}

// GetIndexNodeList node get Index Node List
func GetIndexNodeList() {
	utils.DebugLog("SendMessage(client.IndexNodeConn, req, header.ReqGetIndexNodeList)")
	SendMessageToIndexNodeServer(requests.ReqGetIndexNodeListData(), header.ReqGetIndexNodeList)
}

func SendLatencyCheckMessageToIndexNodeList() {
	utils.DebugLogf("[INDEX_NODE_LATENCY_CHECK] SendHeartbeatToIndexNodeList, num of Index Nodes: %v", len(setting.Config.IndexNodeList))
	if len(setting.Config.IndexNodeList) < 2 {
		utils.ErrorLog("there are not enough Index Nodes in the config file")
		return
	}
	for i := 0; i < len(setting.Config.IndexNodeList); i++ {
		selectedIndexNode := setting.Config.IndexNodeList[i]
		checkSingleIndexNodeLatency(selectedIndexNode.NetworkAddress, false)
	}
}

func checkSingleIndexNodeLatency(server string, heartbeat bool) {
	if client.IndexNodeConn == nil {
		utils.DebugLog("Index Node latency check skipped until connection to Index Node  is recovered")
		return
	}
	utils.DebugLog("[INDEX_NODE_LATENCY_CHECK] SendHeartbeat(", server, ", req, header.ReqHeartbeat)")
	var indexNodeConn *cf.ClientConn
	if client.GetConnectionName(client.IndexNodeConn) != server {
		indexNodeConn = client.NewClient(server, heartbeat)
	} else {
		utils.DebugLog("Checking latency for working Index Node ", server)
		indexNodeConn = client.IndexNodeConn
	}
	//defer indexNodeConn.Close()
	if indexNodeConn != nil {
		start := time.Now().UnixNano()
		pb := &protos.ReqLatencyCheck{
			HbType:                  protos.HeartbeatType_LATENCY_CHECK,
			P2PAddressPp:            setting.P2PAddress,
			NetworkAddressIndexNode: server,
			PingTime:                strconv.FormatInt(start, 10),
		}
		SendMessage(indexNodeConn, pb, header.ReqLatencyCheck)
		if client.GetConnectionName(client.IndexNodeConn) != server {
			bufferedIndexNodeConns = append(bufferedIndexNodeConns, indexNodeConn)
		}
	}
}

func GetBufferedIndexNodeConns() []*cf.ClientConn {
	return bufferedIndexNodeConns
}

func ClearBufferedIndexNodeConns() {
	bufferedIndexNodeConns = make([]*cf.ClientConn, 0)
}

func ScheduleReloadIndexNodelist(future time.Duration) {
	utils.DebugLog("scheduled to get index node list after: ", future.Seconds(), "second")
	ppPeerClock.AddJobWithInterval(future, GetIndexNodeList)
}

func ScheduleReloadPPStatus(future time.Duration) {
	utils.DebugLog("scheduled to get pp status from index node after: ", future.Seconds(), "second")
	ppPeerClock.AddJobWithInterval(future, GetPPStatusInitPPList)
}
