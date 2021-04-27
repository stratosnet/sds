package events

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/utils"
)

type EventHandleFunc func(ctx context.Context, conn spbf.WriteCloser)
type eventCallbackFunc func(ctx context.Context, s *net.Server, message proto.Message, conn spbf.WriteCloser) (proto.Message, string)

const eventHandleErrorTemplate = "event handler error: %s %s: %v"

type event struct {
	version      uint16
	eventType    string
	server       *net.Server
	callbackFunc eventCallbackFunc
}

func newEvent(eventType string, s *net.Server, cb eventCallbackFunc) event {
	return event{
		version:      s.Ver,
		eventType:    eventType,
		server:       s,
		callbackFunc: cb,
	}
}

/*
handle is a generic function that handles all events sent from PP nodes
a callback function and the target (specific event struct) are passed in.
This function is always called in a goroutine, make sure to handle the error message where it is called
go func() {
	err := e.EventHandle(...)
}
*/
func (e *event) handle(
	ctx context.Context,
	conn spbf.WriteCloser,
	target proto.Message,
) error {

	if err := proto.Unmarshal(spbf.MessageFromContext(ctx).MSGData, target); err != nil {
		return fmt.Errorf(eventHandleErrorTemplate, e.eventType, "unmarshal proto message", err)
	}
	utils.DebugLogf("%v receive: %v", e.eventType, target)

	rsp, headerType := e.callbackFunc(ctx, e.server, target, conn)
	if rsp == nil {
		return fmt.Errorf("no response for %v", e.eventType)
	}

	utils.DebugLogf("%v response: %v", e.eventType, utils.ConvertCoronaryUtf8(rsp.(proto.Message).String()))

	data, err := proto.Marshal(rsp.(proto.Message))

	if err != nil {
		return fmt.Errorf("%s response: %v", e.eventType, utils.ConvertCoronaryUtf8(rsp.String()))
	}

	sendBuf := &msg.RelayMsgBuf{
		MSGData: data,
		MSGHead: header.MakeMessageHeader(1, e.version, uint32(len(data)), headerType),
	}

	if err := conn.Write(sendBuf); err != nil {
		return fmt.Errorf(eventHandleErrorTemplate, e.eventType, "write response error", err)
	}

	return nil
}
