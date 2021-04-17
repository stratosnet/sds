package events

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/utils"
)

// transferNotice is a concrete implementation of event
type transferNotice struct {
	event
}

const transferNoticeEvent = "transfer_notice"

// GetTransferNoticeHandler creates event and return handler func for it
func GetTransferNoticeHandler(s *net.Server) EventHandleFunc {
	return transferNotice{
		newEvent(transferNoticeEvent, s, transferNoticeCallbackFunc),
	}.Handle
}

// transferNoticeCallbackFunc is the main process of transfer notice
func transferNoticeCallbackFunc(_ context.Context, _ *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.RspTransferNotice)

	if body.Result.State != protos.ResultState_RES_SUCCESS {

		// todo response failed, prepare another transfer

		utils.ErrorLog(body.TransferCer + ": failed to response to transfer certificate, prepare another transfer")
	}

	return nil, ""
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *transferNotice) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := new(protos.RspTransferNotice)

		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
