package peers

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
)

// SendMessage
func SendMessage(conn core.WriteCloser, pb proto.Message, cmd string) {
	data, err := proto.Marshal(pb)

	if err != nil {
		utils.ErrorLog("error decoding")
		return
	}
	msg := &msg.RelayMsgBuf{
		MSGHead: types.PPMsgHeader(data, cmd),
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
	SendMessageToSPServer(types.ReqGetSPlistData(), header.ReqGetSPList)
}
