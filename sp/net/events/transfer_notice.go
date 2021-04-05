package events

import (
	"context"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/sp/net"
	"github.com/qsnetwork/sds/utils"
)

// TransferNotice
type TransferNotice struct {
	Server *net.Server
}

// GetServer
func (e *TransferNotice) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *TransferNotice) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *TransferNotice) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.RspTransferNotice)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.RspTransferNotice)


		if body.Result.State != protos.ResultState_RES_SUCCESS {

			// todo response failed, prepare another transfer

			utils.Log(body.TransferCer + ": failed to response to transfer certificate, prepare another transfer")
		}

		return nil, ""
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
