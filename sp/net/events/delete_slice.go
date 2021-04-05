package events

import (
	"context"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/sp/net"
	"github.com/qsnetwork/sds/sp/storages/table"
)

// DeleteSlice
type DeleteSlice struct {
	Server *net.Server
}

// GetServer
func (e *DeleteSlice) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *DeleteSlice) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *DeleteSlice) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.RspDeleteSlice)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.RspDeleteSlice)

		fileSlice := new(table.FileSlice)
		fileSlice.SliceHash = body.SliceHash
		fileSlice.WalletAddress = body.WalletAddress

		e.GetServer().CT.Trash(fileSlice)

		return nil, ""
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
