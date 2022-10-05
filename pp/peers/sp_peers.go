package peers

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
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
func SendMessage(ctx context.Context, conn core.WriteCloser, pb proto.Message, cmd string) error {
	data, err := proto.Marshal(pb)

	if err != nil {
		pp.ErrorLog(ctx, "error decoding")
		return errors.New("error decoding")
	}
	msg := &msg.RelayMsgBuf{
		MSGHead: header.MakeMessageHeader(1, uint16(setting.Config.Version.AppVer), uint32(len(data)), cmd),
		MSGData: data,
	}
	switch conn.(type) {
	case *core.ServerConn:
		return conn.(*core.ServerConn).Write(msg, ctx)
	case *cf.ClientConn:
		return conn.(*cf.ClientConn).Write(msg, ctx)
	default:
		return errors.New("unknown connection type")
	}
}

func SendMessageDirectToSPOrViaPP(ctx context.Context, pb proto.Message, cmd string) {
	if client.SPConn != nil {
		SendMessage(ctx, client.SPConn, pb, cmd)
	} else {
		SendMessage(ctx, client.PPConn, pb, cmd)
	}
}

// SendMessageToSPServer SendMessageToSPServer
func SendMessageToSPServer(ctx context.Context, pb proto.Message, cmd string) {
	_, err := ConnectToSP(ctx)
	if err != nil {
		utils.ErrorLog(err)
		return
	}

	SendMessage(ctx, client.SPConn, pb, cmd)
}

// TransferSendMessageToPPServ
func TransferSendMessageToPPServ(ctx context.Context, addr string, msgBuf *msg.RelayMsgBuf) error {
	newCtx := core.CreateContextWithParentReqIdAsReqId(ctx)
	if client.ConnMap[addr] != nil {
		err := client.ConnMap[addr].Write(msgBuf, newCtx)
		utils.DebugLog("conn exist, transfer")
		return err
	}

	utils.DebugLog("new conn, connect and transfer")
	newClient, err := client.NewClient(addr, false)
	if err != nil {
		utils.ErrorLogf("cannot transfer message to client [%v]", addr, utils.FormatError(err))
		return err
	}
	err = newClient.Write(msgBuf, newCtx)
	return err
}

func TransferSendMessageToPPServByP2pAddress(ctx context.Context, p2pAddress string, msgBuf *msg.RelayMsgBuf) {
	ppInfo := peerList.GetPPByP2pAddress(ctx, p2pAddress)
	if ppInfo == nil {
		utils.ErrorLogf("PP %v missing from local ppList. Cannot transfer message due to missing network address", p2pAddress)
		return
	}
	TransferSendMessageToPPServ(ctx, ppInfo.NetworkAddress, msgBuf)
}

// transferSendMessageToSPServer
func TransferSendMessageToSPServer(ctx context.Context, msg *msg.RelayMsgBuf) {
	_, err := ConnectToSP(ctx)
	if err != nil {
		utils.ErrorLog(err)
		return
	}

	client.SPConn.Write(msg, ctx)
}

// ReqTransferSendSP
func ReqTransferSendSP(ctx context.Context, conn core.WriteCloser) {
	TransferSendMessageToSPServer(ctx, core.MessageFromContext(ctx))
}

// transferSendMessageToClient
func TransferSendMessageToClient(ctx context.Context, p2pAddress string, msgBuf *msg.RelayMsgBuf) {
	ppNode := peerList.GetPPByP2pAddress(ctx, p2pAddress)
	if ppNode != nil && ppNode.Status == types.PEER_CONNECTED {
		pp.Log(ctx, "transfer to netid = ", ppNode.NetId)
		GetPPServer().Unicast(ctx, ppNode.NetId, msgBuf)
	} else {
		pp.DebugLog(ctx, "waller ===== ", p2pAddress)
	}
}

// GetMyNodeStatusFromSP P node get node status
func GetPPStatusFromSP(ctx context.Context) {
	pp.DebugLog(ctx, "SendMessage(client.SPConn, req, header.ReqGetPPStatus)")
	SendMessageToSPServer(ctx, requests.ReqGetPPStatusData(false), header.ReqGetPPStatus)
}

// GetMyNodeStatusFromSP P node get node status
func GetPPStatusInitPPList(ctx context.Context) func() {
	return func() {
		pp.DebugLog(ctx, "SendMessage(client.SPConn, req, header.ReqGetPPStatus)")
		SendMessageToSPServer(ctx, requests.ReqGetPPStatusData(true), header.ReqGetPPStatus)
	}
}

// GetSPList node get spList
func GetSPList(ctx context.Context) func() {
	return func() {
		pp.DebugLog(ctx, "SendMessage(client.SPConn, req, header.ReqGetSPList)")
		SendMessageToSPServer(ctx, requests.ReqGetSPlistData(), header.ReqGetSPList)
	}
}

func SendLatencyCheckMessageToSPList(ctx context.Context) {
	utils.DebugLogf("[SP_LATENCY_CHECK] SendHeartbeatToSPList, num of SPs: %v", len(setting.Config.SPList))
	if len(setting.Config.SPList) < 2 {
		utils.ErrorLog("there are not enough SP nodes in the config file")
		return
	}
	for i := 0; i < len(setting.Config.SPList); i++ {
		selectedSP := setting.Config.SPList[i]
		checkSingleSpLatency(ctx, selectedSP.NetworkAddress, false)
	}
}

func checkSingleSpLatency(ctx context.Context, server string, heartbeat bool) {
	if client.SPConn == nil {
		utils.DebugLog("SP latency check skipped until connection to SP is recovered")
		return
	}
	utils.DebugLog("[SP_LATENCY_CHECK] SendHeartbeat(", server, ", req, header.ReqHeartbeat)")
	var spConn *cf.ClientConn
	var err error
	if client.GetConnectionName(client.SPConn) != server {
		spConn, err = client.NewClient(server, heartbeat)
		if err != nil {
			utils.DebugLogf("failed to connect to server %v: %v", server, utils.FormatError(err))
		}
	} else {
		utils.DebugLog("Checking latency for working SP ", server)
		spConn = client.SPConn
	}
	//defer spConn.Close()
	if spConn != nil {
		start := time.Now().UnixNano()
		pb := &protos.ReqLatencyCheck{
			HbType:           protos.HeartbeatType_LATENCY_CHECK,
			P2PAddressPp:     setting.P2PAddress,
			NetworkAddressSp: server,
			PingTime:         strconv.FormatInt(start, 10),
		}
		SendMessage(ctx, spConn, pb, header.ReqLatencyCheck)
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

func ScheduleReloadSPlist(ctx context.Context, future time.Duration) {
	utils.DebugLog("scheduled to get sp-list after: ", future.Seconds(), "second")
	ppPeerClock.AddJobWithInterval(future, GetSPList(ctx))
}

func ScheduleReloadPPStatus(ctx context.Context, future time.Duration) {
	utils.DebugLog("scheduled to get pp status from sp after: ", future.Seconds(), "second")
	ppPeerClock.AddJobWithInterval(future, GetPPStatusInitPPList(ctx))
}

// CreateNewContextPacketId used for downloading / uploading speed tracking
func CreateNewContextPacketId(ctx context.Context) (int64, context.Context) {
	retCtx := ctx
	packetId, _ := utils.NextSnowFlakeId()
	utils.DebugLogf("PacketId in new context: %v", strconv.FormatInt(packetId, 10))
	return packetId, core.CreateContextWithPacketId(retCtx, packetId)
}
