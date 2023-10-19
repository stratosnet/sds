package p2pserver

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
	utilstypes "github.com/stratosnet/sds/utils/types"
)

var (
	requestInfoMap = utils.NewAutoCleanMap(60 * time.Minute) // used for req-rsp message pair verifications
)

func (p *P2pServer) SignP2pMessage(signMsg []byte) []byte {
	return p.p2pPrivKey.Sign(signMsg)
}

func (p *P2pServer) GetP2PPublicKey() []byte {
	return p.p2pPubKey.Bytes()
}

func (p *P2pServer) GetP2PAddress() string {
	addr, err := p.p2pAddress.P2pAddressToBech()
	if err != nil {
		return ""
	}
	return addr
}

func (p *P2pServer) GetPPInfo() *protos.PPBaseInfo {
	return &protos.PPBaseInfo{
		P2PAddress:     p.GetP2PAddress(),
		WalletAddress:  setting.WalletAddress,
		NetworkAddress: setting.NetworkAddress,
		RestAddress:    setting.RestAddress,
	}
}

func (p *P2pServer) LoadRequestInfo(reqId int64, rspMsgType uint8) (uint8, bool) {
	// according to the rsp, load the request info by (reqId | supposed_req_msg_type)
	msgTypeId, found := requestInfoMap.Load(reqId&0x7FFFFFFFFFFFFF00 | int64(header.GetReqIdFromRspId(rspMsgType)))
	if !found {
		return header.MSG_ID_INVALID, found
	}
	return msgTypeId.(uint8), found
}

func (p *P2pServer) StoreRequestInfo(reqId int64, reqMsgType uint8) {
	// reqId includes the msg type of original request. The consequent requests need to be re-encoded as the index for requestInfoMap
	requestInfoMap.Store((reqId&0x7FFFFFFFFFFFFF00)|int64(reqMsgType), reqMsgType)
}

func (p *P2pServer) GetP2PAddrInTypeAddress() utilstypes.Address {
	return p.p2pAddress
}

func (p *P2pServer) SendMessage(ctx context.Context, conn core.WriteCloser, pb proto.Message, cmd header.MsgType) error {
	msgBuf := &msg.RelayMsgBuf{
		MSGSign: msg.MessageSign{
			P2pPubKey:  p.GetP2PPublicKey(),
			P2pAddress: p.GetP2PAddress(),
			Signer:     p.SignP2pMessage,
		},
	}

	switch cmd.Id {
	case header.MSG_ID_REQ_UPLOAD_FILESLICE:
		msgBuf.MSGData = pb.(*protos.ReqUploadFileSlice).Data
		pb.(*protos.ReqUploadFileSlice).Data = nil
	case header.MSG_ID_REQ_BACKUP_FILESLICE:
		msgBuf.MSGData = pb.(*protos.ReqBackupFileSlice).Data
		pb.(*protos.ReqBackupFileSlice).Data = nil
	case header.MSG_ID_RSP_DOWNLOAD_SLICE:
		msgBuf.MSGData = pb.(*protos.RspDownloadSlice).Data
		pb.(*protos.RspDownloadSlice).Data = nil
	case header.MSG_ID_RSP_TRANSFER_DOWNLOAD:
		msgBuf.MSGData = pb.(*protos.RspTransferDownload).Data
		pb.(*protos.RspTransferDownload).Data = nil
	}

	if strings.HasPrefix(cmd.Name, "Req") {
		reqId := core.GetReqIdFromContext(ctx)
		if reqId == 0 {
			reqId = core.GenerateNewReqId(cmd.Id)
			core.InheritRpcLoggerFromParentReqId(ctx, reqId)
			core.InheritRemoteReqIdFromParentReqId(ctx, reqId)
		}
		msgBuf.MSGHead.ReqId = reqId
		p.StoreRequestInfo(reqId, cmd.Id)
	}

	body, err := proto.Marshal(pb)
	if err != nil {
		pp.ErrorLog(ctx, "error decoding")
		return errors.New("error decoding")
	}
	msgBuf.MSGBody = body
	reqId := msgBuf.MSGHead.ReqId
	msgBuf.MSGHead = header.MakeMessageHeader(1, setting.Config.Version.AppVer, uint32(len(body)), cmd)
	msgBuf.MSGHead.ReqId = reqId
	switch conn := conn.(type) {
	case *core.ServerConn:
		return conn.Write(msgBuf, ctx)
	case *cf.ClientConn:
		return conn.Write(msgBuf, ctx)
	default:
		return errors.New("unknown connection type")
	}
}

func (p *P2pServer) SendMessageDirectToSPOrViaPP(ctx context.Context, pb proto.Message, cmd header.MsgType) {
	if p.mainSpConn != nil {
		_ = p.SendMessage(ctx, p.mainSpConn, pb, cmd)
	} else {
		_ = p.SendMessage(ctx, p.ppConn, pb, cmd)
	}
}

func (p *P2pServer) SendMessageToSPServer(ctx context.Context, pb proto.Message, cmd header.MsgType) {
	if p.mainSpConn != nil {
		_ = p.SendMessage(ctx, p.mainSpConn, pb, cmd)
	}
}

func (p *P2pServer) TransferSendMessageToPPServ(ctx context.Context, addr string, msgBuf *msg.RelayMsgBuf) error {
	newCtx := core.CreateContextWithParentReqIdAsReqId(ctx)
	cmd := header.GetMsgTypeFromId(msgBuf.MSGHead.Cmd)
	if cmd == nil {
		return errors.New(fmt.Sprintf("invalid message type %d", msgBuf.MSGHead.Cmd))
	}
	msgBuf.MSGSign = msg.MessageSign{
		P2pPubKey:  p.GetP2PPublicKey(),
		P2pAddress: p.GetP2PAddress(),
		Signer:     p.SignP2pMessage,
	}
	if strings.HasPrefix(cmd.Name, "Req") {
		reqId := core.GetReqIdFromContext(ctx)
		if reqId == 0 {
			reqId = core.GenerateNewReqId(msgBuf.MSGHead.Cmd)
			core.InheritRpcLoggerFromParentReqId(ctx, reqId)
			core.InheritRemoteReqIdFromParentReqId(ctx, reqId)
		}
		msgBuf.MSGHead.ReqId = reqId
		p.StoreRequestInfo(reqId, cmd.Id)
	}

	p.clientMutex.Lock()
	if p.connMap[addr] != nil {
		err := p.connMap[addr].Write(msgBuf, newCtx)
		p.clientMutex.Unlock()
		if err != nil {
			utils.DebugLogf("Error writing msg to %s, %v", addr, err.Error())
		}
		return err
	}
	p.clientMutex.Unlock()

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
		P2pPubKey:  p.GetP2PPublicKey(),
		P2pAddress: p.GetP2PAddress(),
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

func (p *P2pServer) setWriteHook(conn *cf.ClientConn, callback core.WriteHookFunc) {
	if conn != nil {
		var hooks []cf.WriteHook
		hook := cf.WriteHook{
			MessageId: header.ReqUploadFileSlice.Id,
			Fn:        callback,
		}
		hooks = append(hooks, hook)
		conn.SetWriteHook(hooks)
	}
}

func (p *P2pServer) SendMessageByCachedConn(ctx context.Context, key string, netAddr string, pb proto.Message, cmd header.MsgType, fn core.WriteHookFunc) error {
	// use the cached conn to send the message
	if conn, ok := p.LoadConnFromCache(key); ok {
		if fn != nil {
			p.setWriteHook(conn, fn)
		}
		err := p.SendMessage(ctx, conn, pb, cmd)
		if err == nil {
			utils.DebugLog("SendMessage(conn, pb, header.", cmd.Name, ") ", conn)
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
		utils.DebugLog("SendMessage(conn, pb, header.ReqUploadFileSlice) ", conn)
		p.StoreConnToCache(key, conn)
	} else {
		utils.ErrorLog("Fail to send upload slice request to " + netAddr + ", " + err.Error())
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
