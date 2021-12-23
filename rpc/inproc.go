package rpc

import (
	"context"
	"net"
)

// DialInProc attaches an in-process connection to the given RPC server.
func DialInProc(handler *Server) *Client {
	initctx := context.Background()
	c, _ := newClient(initctx, func(context.Context) (ServerCodec, error) {
		p1, p2 := net.Pipe()
		go handler.ServeCodec(NewCodec(p1), 0)
		return NewCodec(p2), nil
	})
	return c
}
