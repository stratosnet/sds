package net

import (
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/utils"
)

/*


DEPRECATED


*/
import (
	"context"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"reflect"
)

// Event
// @notice
type Event interface {
	GetServer() *Server
	SetServer(server *Server)
	Handle(ctx context.Context, conn spbf.WriteCloser)
}

/*
EventHandle is a generic function that handle all events sent from PP nodes
a callback function and the target (specific event struct) are passed in.
This function is always called in a goroutine, make sure to handle the error message where it is called
go func() {
	err := e.EventHandle(...)
}
*/
func EventHandle(
	ctx context.Context,
	conn spbf.WriteCloser,
	target interface{},
	callback func(message interface{}) (interface{}, string),
	version uint16,
) error {

	msgBuf := spbf.MessageFromContext(ctx)
	if utils.CheckError(proto.Unmarshal(msgBuf.MSGData, target.(proto.Message))) {
		return errors.New("unmarshal proto fail")
	}

	fmt.Println("#####", reflect.TypeOf(target), "===============================")
	fmt.Println()
	fmt.Println("##### receive: ")
	fmt.Println()
	fmt.Println(target)
	fmt.Println()

	rsp, headerType := callback(target)

	if rsp != nil {

		fmt.Println("##### response: ")
		fmt.Println()
		fmt.Println(utils.ConvertCoronaryUtf8(rsp.(proto.Message).String()))
		fmt.Println()

		data, err := proto.Marshal(rsp.(proto.Message))

		if err != nil {
			return err
		}

		sendBuf := &msg.RelayMsgBuf{
			MSGData: data,
			MSGHead: header.MakeMessageHeader(1, version, uint32(len(data)), headerType),
		}

		conn.Write(sendBuf)
	}

	return nil
}
