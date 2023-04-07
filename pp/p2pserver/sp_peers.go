package p2pserver

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
	utiltypes "github.com/stratosnet/sds/utils/types"
	"google.golang.org/protobuf/proto"
)

func (p *P2pServer) SignP2pMessage(signMsg []byte) []byte {
	return utiltypes.BytesToP2pPrivKey(setting.P2PPrivateKey).Sign(signMsg)
}

func (p *P2pServer) SendMessage(ctx context.Context, conn core.WriteCloser, pb proto.Message, cmd string) error {
	msg := &msg.RelayMsgBuf{
		MSGSign: msg.MessageSign{
			P2pPubKey:  setting.P2PPublicKey,
			P2pAddress: setting.P2PAddress,
			Signer:     p.SignP2pMessage,
		},
	}

	switch cmd {
	case header.ReqUploadFileSlice:
		msg.MSGData = pb.(*protos.ReqUploadFileSlice).Data
		pb.(*protos.ReqUploadFileSlice).Data = nil
	case header.ReqBackupFileSlice:
		msg.MSGData = pb.(*protos.ReqBackupFileSlice).Data
		pb.(*protos.ReqBackupFileSlice).Data = nil
	case header.RspDownloadSlice:
		msg.MSGData = pb.(*protos.RspDownloadSlice).Data
		pb.(*protos.RspDownloadSlice).Data = nil
	case header.RspTransferDownload:
		msg.MSGData = pb.(*protos.RspTransferDownload).Data
		pb.(*protos.RspTransferDownload).Data = nil
	}

	msg.MSGHead.DataLen = uint32(len(msg.MSGData))
	body, err := proto.Marshal(pb)
	if err != nil {
		pp.ErrorLog(ctx, "error decoding")
		return errors.New("error decoding")
	}
	msg.MSGBody = body
	msg.MSGHead = header.MakeMessageHeader(1, setting.Config.Version.AppVer, uint32(len(body)), cmd)
	switch conn := conn.(type) {
	case *core.ServerConn:
		return conn.Write(msg, ctx)
	case *cf.ClientConn:
		return conn.Write(msg, ctx)
	default:
		return errors.New("unknown connection type")
	}
}

func (p *P2pServer) SendMessageDirectToSPOrViaPP(ctx context.Context, pb proto.Message, cmd string) {
	if p.mainSpConn != nil {
		_ = p.SendMessage(ctx, p.mainSpConn, pb, cmd)
	} else {
		_ = p.SendMessage(ctx, p.ppConn, pb, cmd)
	}
}

func (p *P2pServer) SendMessageToSPServer(ctx context.Context, pb proto.Message, cmd string) {
	if p.mainSpConn != nil {
		_ = p.SendMessage(ctx, p.mainSpConn, pb, cmd)
	}
}

func (p *P2pServer) TransferSendMessageToPPServ(ctx context.Context, addr string, msgBuf *msg.RelayMsgBuf) error {
	newCtx := core.CreateContextWithParentReqIdAsReqId(ctx)
	msgBuf.MSGSign = msg.MessageSign{
		P2pPubKey:  setting.P2PPublicKey,
		P2pAddress: setting.P2PAddress,
		Signer:     p.SignP2pMessage,
	}

	if p.connMap[addr] != nil {
		err := p.connMap[addr].Write(msgBuf, newCtx)
		if err != nil {
			utils.DebugLogf("Error writing msg to %s, %v", addr, err.Error())
		}
		return err
	}

	utils.DebugLog("new conn, connect and transfer")
	newClient, err := p.NewClientToPp(ctx, addr, false)
	if err != nil {
		utils.ErrorLogf("cannot transfer message to client [%v]: %v", addr, utils.FormatError(err))
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
	_ = p.TransferSendMessageToPPServ(ctx, ppInfo.NetworkAddress, msgBuf)
}

func (p *P2pServer) TransferSendMessageToSPServer(ctx context.Context, message *msg.RelayMsgBuf) {
	_, err := p.ConnectToSP(ctx)
	if err != nil {
		utils.ErrorLog(err)
		return
	}
	message.MSGSign = msg.MessageSign{
		P2pPubKey:  setting.P2PPublicKey,
		P2pAddress: setting.P2PAddress,
		Signer:     p.SignP2pMessage,
	}

	_ = p.mainSpConn.Write(message, ctx)
}

func (p *P2pServer) ReqTransferSendSP(ctx context.Context, conn core.WriteCloser) {
	p.TransferSendMessageToSPServer(ctx, core.MessageFromContext(ctx))
}

func (p *P2pServer) TransferSendMessageToClient(ctx context.Context, p2pAddress string, msgBuf *msg.RelayMsgBuf) {
	ppNode := p.peerList.GetPPByP2pAddress(ctx, p2pAddress)
	if ppNode != nil && ppNode.Status == types.PEER_CONNECTED {
		pp.Log(ctx, "transfer to netid = ", ppNode.NetId)
		_ = p.GetP2pServer().Unicast(ctx, ppNode.NetId, msgBuf)
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
