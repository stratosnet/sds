package core

//event register
import (
	"context"

	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/utils"
)

type ctxkey string

const (
	messageCtxKey     ctxkey = "message"
	serverCtxKey      ctxkey = "server"
	netIDCtxKey       ctxkey = "netid"
	reqIDCtxKey       ctxkey = "reqId"
	parentReqIDCtxKey ctxkey = "parentReqId"
)

var (
	messageRegistry map[string]HandlerFunc
)

func init() {
	messageRegistry = map[string]HandlerFunc{}
}

// Register
func Register(cmd string, handler func(context.Context, WriteCloser)) {
	if _, ok := messageRegistry[cmd]; ok {
	}
	messageRegistry[cmd] = handler
}

// Handler
type Handler interface {
	Handle(context.Context, interface{})
}

// HandlerFunc
type HandlerFunc func(context.Context, WriteCloser)

// Handle
func (f HandlerFunc) Handle(ctx context.Context, c WriteCloser) {
	f(ctx, c)
}

// GetHandlerFunc
func GetHandlerFunc(msgType string) HandlerFunc {
	entry, ok := messageRegistry[msgType]
	if !ok {
		return nil
	}
	return entry
}

// CreateContextWithMessage
func CreateContextWithMessage(ctx context.Context, message *msg.RelayMsgBuf) context.Context {
	return context.WithValue(ctx, messageCtxKey, message)
}

// MessageFromContext get msg from context
func MessageFromContext(ctx context.Context) *msg.RelayMsgBuf {
	return ctx.Value(messageCtxKey).(*msg.RelayMsgBuf)
}

// CreateContextWithNetID
func CreateContextWithNetID(ctx context.Context, netID int64) context.Context {
	return context.WithValue(ctx, netIDCtxKey, netID)
}

// NetIDFromContext
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

func GetReqIdFromContext(ctx context.Context) int64 {
	if ctx == nil || ctx.Value(reqIDCtxKey) == nil {
		return 0
	}

	reqId := ctx.Value(reqIDCtxKey).(int64)
	return reqId
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
