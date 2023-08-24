package core

//event register
import (
	"context"
	"time"

	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/utils"
)

type ctxkey string

const (
	messageCtxKey     ctxkey = "message"
	serverCtxKey      ctxkey = "server"
	netIDCtxKey       ctxkey = "netid"
	reqIDCtxKey       ctxkey = "reqId"
	packetIDCtxKey    ctxkey = "packetId"
	parentReqIDCtxKey ctxkey = "parentReqId"
	recvStartKey      ctxkey = "recvStartTime"
	srcP2pAddrCtxKey  ctxkey = "srcP2pAddr"
)

var (
	messageRegistry [header.NUMBER_MESSAGE_TYPES]HandlerFunc
)

func Register(cmd header.MsgType, handler func(context.Context, WriteCloser)) {
	messageRegistry[cmd.Id] = handler
}

type Handler interface {
	Handle(context.Context, interface{})
}

type HandlerFunc func(context.Context, WriteCloser)

func (f HandlerFunc) Handle(ctx context.Context, c WriteCloser) {
	f(ctx, c)
}

func GetHandlerFunc(id uint8) HandlerFunc {
	if id >= header.NUMBER_MESSAGE_TYPES {
		return nil
	}
	return messageRegistry[id]
}

func CreateContextWithMessage(ctx context.Context, message *msg.RelayMsgBuf) context.Context {
	return context.WithValue(ctx, messageCtxKey, message)
}

// MessageFromContext get msg from context
func MessageFromContext(ctx context.Context) *msg.RelayMsgBuf {
	return ctx.Value(messageCtxKey).(*msg.RelayMsgBuf)
}

func CreateContextWithNetID(ctx context.Context, netID int64) context.Context {
	return context.WithValue(ctx, netIDCtxKey, netID)
}

func NetIDFromContext(ctx context.Context) int64 {
	return ctx.Value(netIDCtxKey).(int64)
}

func InheritRpcLoggerFromParentReqId(ctx context.Context, reqId int64) {
	if ctx == nil || ctx.Value(parentReqIDCtxKey) == nil {
		return
	}
	parentReqId := ctx.Value(parentReqIDCtxKey).(int64)
	utils.RegisterReqToParentReq(reqId, parentReqId)
}

func GetPacketIdFromContext(ctx context.Context) int64 {
	if ctx == nil || ctx.Value(packetIDCtxKey) == nil {
		return 0
	}

	packetId := ctx.Value(packetIDCtxKey).(int64)
	return packetId
}

func GetReqIdFromContext(ctx context.Context) int64 {
	if ctx == nil || ctx.Value(reqIDCtxKey) == nil {
		return 0
	}

	reqId := ctx.Value(reqIDCtxKey).(int64)
	return reqId
}

func GetRecvStartTimeFromContext(ctx context.Context) int64 {
	if ctx == nil || ctx.Value(recvStartKey) == nil {
		return 0
	}

	recvStart := ctx.Value(recvStartKey).(int64)
	return recvStart
}

func GetRecvCostTimeFromContext(ctx context.Context) int64 {
	if ctx == nil || ctx.Value(recvStartKey) == nil {
		return 0
	}
	now := time.Now().UnixMilli()
	recvStart := ctx.Value(recvStartKey).(int64)
	return now - recvStart
}

func GetParentReqIdFromContext(ctx context.Context) int64 {
	if ctx == nil || ctx.Value(parentReqIDCtxKey) == nil {
		return 0
	}

	parentReqId := ctx.Value(parentReqIDCtxKey).(int64)
	return parentReqId
}

func GetSrcP2pAddrFromContext(ctx context.Context) string {
	if ctx == nil || ctx.Value(srcP2pAddrCtxKey) == nil {
		return ""
	}

	srcP2pAddress := ctx.Value(srcP2pAddrCtxKey).(string)
	return srcP2pAddress
}

func CreateContextWithReqId(ctx context.Context, reqId int64) context.Context {
	return context.WithValue(ctx, reqIDCtxKey, reqId)
}

func CreateContextWithPacketId(ctx context.Context, packetId int64) context.Context {
	return context.WithValue(ctx, packetIDCtxKey, packetId)
}

func CreateContextWithRecvStartTime(ctx context.Context, recvStartTime int64) context.Context {
	return context.WithValue(ctx, recvStartKey, recvStartTime)
}

func CreateContextWithParentReqId(ctx context.Context, reqId int64) context.Context {
	return context.WithValue(ctx, parentReqIDCtxKey, reqId)
}

func CreateContextWithParentReqIdAsReqId(ctx context.Context) context.Context {
	if ctx != nil && ctx.Value(parentReqIDCtxKey) != nil {
		parentReqId := ctx.Value(parentReqIDCtxKey).(int64)
		return context.WithValue(ctx, reqIDCtxKey, parentReqId)
	}
	return ctx
}

func CreateContextWithSrcP2pAddr(ctx context.Context, srcP2pAddress string) context.Context {
	return context.WithValue(ctx, srcP2pAddrCtxKey, srcP2pAddress)
}

func reqIdFromSnowFlake(snowflake int64, msgid uint8) int64 {
	// lsb of snowflake is replaced by msg id without losing the uniqueness. There are still 8 bits in the
	// second-lowest byte for sequence number.
	return int64((uint64(snowflake) & 0xFFFFFFFFFFFF0000) | (uint64(snowflake) & 0xFF << 8) | uint64(msgid))
}

func GenerateNewReqId(msgid uint8) int64 {
	snowFlake, err := utils.NextSnowFlakeId()
	if err != nil {
		utils.FatalLogfAndExit(-3, "Fatal error: "+err.Error())
	}
	return reqIdFromSnowFlake(snowFlake, msgid)
}
