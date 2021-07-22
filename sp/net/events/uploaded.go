package events

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/utils"
)

// uploaded is a concrete implementation of event
// stratoschain FileUpload transaction success
type uploaded struct {
	event
}

const uploadedEvent = "uploaded"

// GetUploadedHandler creates event and return handler func for it
func GetUploadedHandler(s *net.Server) EventHandleFunc {
	e := uploaded{newEvent(uploadedEvent, s, uploadedCallbackFunc)}
	return e.Handle
}

// uploadedCallbackFunc is the main process of FileUploadTx being successful
func uploadedCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.Uploaded)
	fmt.Printf("File %v was successfully uploaded by node %v (reporter is %v)\n", body.FileHash, body.UploaderAddress, body.ReporterAddress)
	// TODO: Update the traffic record, the index, and start replicating tasks here

	return nil, ""
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *uploaded) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.Uploaded{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
