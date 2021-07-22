package events

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
)

// deleteSlice is a concrete implementation of event
type deleteSlice struct {
	event
}

const deleteSliceEvent = "delete_slice"

// GetDeleteSliceHandler creates event and return handler func for it
func GetDeleteSliceHandler(s *net.Server) EventHandleFunc {
	e := deleteSlice{newEvent(deleteSliceEvent, s, deleteSliceCallbackFunc)}
	return e.Handle
}

// deleteSliceCallbackFunc is the main process of delete slice event
func deleteSliceCallbackFunc(ctx context.Context, s *net.Server, message proto.Message, conn spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.RspDeleteSlice)

	fileSlice := &table.FileSlice{
		SliceHash: body.SliceHash,
		FileSliceStorage: table.FileSliceStorage{
			P2pAddress: body.P2PAddress,
		},
	}

	if err := s.CT.Trash(fileSlice); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, deleteSliceEvent, "trash file slice from db", err)
	}

	return nil, ""
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *deleteSlice) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.RspDeleteSlice{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
