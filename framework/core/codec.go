package core

//event register
import (
	"context"
	"github.com/stratosnet/sds/msg"
)

type ctxkey string

const (
	messageCtxKey ctxkey = "message"
	serverCtxKey  ctxkey = "server"
	netIDCtxKey   ctxkey = "netid"
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
