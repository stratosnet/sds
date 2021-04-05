package main

import (
	"context"
	"fmt"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg/header"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/pp/serv"
	"github.com/qsnetwork/sds/sp/net"
)

func main() {

	spbf.Register(header.ReqRegisterNewPP, new(MyHandle).Handle)

	// spbf.Register ...

	serv.StartListenServer(":8888")
}


type MyHandle struct {
	Server *net.Server
}

func (t *MyHandle) GetServer() *net.Server {
	return t.Server
}

func (t *MyHandle) SetServer(server *net.Server) {
	t.Server = server
}

func (t *MyHandle) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqRegisterNewPP)

	callback := func(message interface{}) (interface{}, string) {

		// body := message.(*protos.ReqRegisterNewPP)

		fmt.Println("handle...")

		// coding...

		rsp := &protos.RspRegisterNewPP{
			Result: &protos.Result{State: protos.ResultState_RES_SUCCESS},
		}

		return rsp, header.RspRegisterNewPP
	}

	net.EventHandle(ctx, conn, target, callback, 1)
}
