package net

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/utils"
	"reflect"
)

// Event
// @notice
type Event interface {
	GetServer() *Server
	SetServer(server *Server)
	Handle(ctx context.Context, conn spbf.WriteCloser)
}

// EventHandle
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
