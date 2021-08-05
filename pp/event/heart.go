package event

import (
	"context"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
)

// SendHeartBeat
func SendHeartBeat(ctx context.Context, conn core.WriteCloser) {
	// utils.DebugLog("send HeartBeat")
	msg := msg.RelayMsgBuf{
		MSGHead: PPMsgHeader(nil, header.RspHeart),
	}
	conn.Write(&msg)
}

// RspHeartBeat
func RspHeartBeat(ctx context.Context, conn core.WriteCloser) {
	// utils.DebugLog("ResHeartBeat")
	switch conn.(type) {
	case *core.ServerConn:
		msg := msg.RelayMsgBuf{
			MSGHead: PPMsgHeader(nil, header.RspHeart),
		}
		conn.Write(&msg)
	}
}
