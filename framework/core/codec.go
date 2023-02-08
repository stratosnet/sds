package core

//event register
import (
	"context"
	"time"

	"github.com/stratosnet/sds/msg"
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
)

var (
	messageRegistry map[string]HandlerFunc
)

func init() {
	messageRegistry = map[string]HandlerFunc{}
}

func Register(cmd string, handler func(context.Context, WriteCloser)) {
	messageRegistry[cmd] = handler
}

type Handler interface {
	Handle(context.Context, interface{})
}

type HandlerFunc func(context.Context, WriteCloser)

func (f HandlerFunc) Handle(ctx context.Context, c WriteCloser) {
	f(ctx, c)
}

func GetHandlerFunc(msgType string) HandlerFunc {
	entry, ok := messageRegistry[msgType]
	if !ok {
		return nil
	}
	return entry
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
	if logger, ok := utils.RpcLoggerMap.Load(parentReqId); ok {
		utils.RpcLoggerMap.Store(reqId, logger)
	}
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
