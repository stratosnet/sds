package p2pserver

import (
	"context"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/utils"
)

// SendMessage
func (p *P2pServer) SendMessage(ctx context.Context, conn core.WriteCloser, pb proto.Message, cmd string) error {
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

func (p *P2pServer) SendMessageDirectToSPOrViaPP(ctx context.Context, pb proto.Message, cmd string) {
	if p.mainSpConn != nil {
		p.SendMessage(ctx, p.mainSpConn, pb, cmd)
	} else {
		p.SendMessage(ctx, p.ppConn, pb, cmd)
	}
}

// SendMessageToSPServer SendMessageToSPServer
func (p *P2pServer) SendMessageToSPServer(ctx context.Context, pb proto.Message, cmd string) {
	//_, err := p.ConnectToSP(ctx)
	//if err != nil {
	//	utils.ErrorLog(err)
	//	return
	//}
	if p.mainSpConn != nil {
		p.SendMessage(ctx, p.mainSpConn, pb, cmd)
	}
}

// TransferSendMessageToPPServ
func (p *P2pServer) TransferSendMessageToPPServ(ctx context.Context, addr string, msgBuf *msg.RelayMsgBuf) error {
	newCtx := core.CreateContextWithParentReqIdAsReqId(ctx)
	//p.ClientMutex.Lock()
	if p.connMap[addr] != nil {
		err := p.connMap[addr].Write(msgBuf, newCtx)
		//p.ClientMutex.Unlock()
		utils.DebugLog("conn exist, transfer")
		return err
	}

	utils.DebugLog("new conn, connect and transfer")
	newClient, err := p.NewClientToPp(ctx, addr, false)
	if err != nil {
		utils.ErrorLogf("cannot transfer message to client [%v]", addr, utils.FormatError(err))
		return err
	}
	err = newClient.Write(msgBuf, newCtx)
	return err
}

func (p *P2pServer) TransferSendMessageToPPServByP2pAddress(ctx context.Context, p2pAddress string, msgBuf *msg.RelayMsgBuf) {
	ppInfo := p.peerList.GetPPByP2pAddress(ctx, p2pAddress)
	if ppInfo == nil {
		utils.ErrorLogf("PP %v missing from local ppList. Cannot transfer message due to missing network address", p2pAddress)
		return
	}
	p.TransferSendMessageToPPServ(ctx, ppInfo.NetworkAddress, msgBuf)
}

// transferSendMessageToSPServer
func (p *P2pServer) TransferSendMessageToSPServer(ctx context.Context, msg *msg.RelayMsgBuf) {
	_, err := p.ConnectToSP(ctx)
	if err != nil {
		utils.ErrorLog(err)
		return
	}

	p.mainSpConn.Write(msg, ctx)
}

// ReqTransferSendSP
func (p *P2pServer) ReqTransferSendSP(ctx context.Context, conn core.WriteCloser) {
	p.TransferSendMessageToSPServer(ctx, core.MessageFromContext(ctx))
}

// transferSendMessageToClient
func (p *P2pServer) TransferSendMessageToClient(ctx context.Context, p2pAddress string, msgBuf *msg.RelayMsgBuf) {
	ppNode := p.peerList.GetPPByP2pAddress(ctx, p2pAddress)
	if ppNode != nil && ppNode.Status == types.PEER_CONNECTED {
		pp.Log(ctx, "transfer to netid = ", ppNode.NetId)
		p.GetP2pServer().Unicast(ctx, ppNode.NetId, msgBuf)
	} else {
		pp.DebugLog(ctx, "waller ===== ", p2pAddress)
	}
}

func (p *P2pServer) GetBufferedSpConns() []*cf.ClientConn {
	return p.bufferedSpConns
}

func (p *P2pServer) StoreBufferedSpConn(spConn *cf.ClientConn) {
	p.bufferedSpConns = append(p.bufferedSpConns, spConn)
}

func (p *P2pServer) ClearBufferedSpConns() {
	p.bufferedSpConns = make([]*cf.ClientConn, 0)
}

func (p *P2pServer) setWriteHook(conn *cf.ClientConn, callback func(packetId, costTime int64)) {
	if conn != nil {
		var hooks []cf.WriteHook
		hook := cf.WriteHook{
			Message: header.ReqUploadFileSlice,
			Fn:      callback,
		}
		hooks = append(hooks, hook)
		conn.SetWriteHook(hooks)
	}
}

func (p *P2pServer) SendMessageByCachedConn(ctx context.Context, key string, netAddr string, pb proto.Message, cmd string, fn func(packetId, costTime int64)) error {
	// use the cached conn to send the message
	if conn, ok := p.LoadConnFromCache(key); ok {
		if fn != nil {
			p.setWriteHook(conn, fn)
		}
		err := p.SendMessage(ctx, conn, pb, cmd)
		if err == nil {
			pp.DebugLog(ctx, "SendMessage(conn, pb, header.ReqUploadFileSlice) ", conn)
			return err
		}
	}
	// not in cache, connect to the network address
	conn, err := p.NewClientToPp(ctx, netAddr, false)
	if err != nil {
		return errors.Wrap(err, "Failed to create connection with "+netAddr)
	}
	if fn != nil {
		p.setWriteHook(conn, fn)
	}
	err = p.SendMessage(ctx, conn, pb, cmd)
	if err == nil {
		pp.DebugLog(ctx, "SendMessage(conn, pb, header.ReqUploadFileSlice) ", conn)
		p.StoreConnToCache(key, conn)
	} else {
		pp.ErrorLog(ctx, "Fail to send upload slice request to "+netAddr)
	}
	return err
}

// CreateNewContextPacketId used for downloading / uploading speed tracking
func CreateNewContextPacketId(ctx context.Context) (int64, context.Context) {
	retCtx := ctx
	packetId, _ := utils.NextSnowFlakeId()
	utils.DebugLogf("PacketId in new context: %v", strconv.FormatInt(packetId, 10))
	return packetId, core.CreateContextWithPacketId(retCtx, packetId)
}
