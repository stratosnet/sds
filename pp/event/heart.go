package event



import (
	"context"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg"
	"github.com/qsnetwork/sds/msg/header"
)

// SendHeartBeat
func SendHeartBeat(ctx context.Context, conn spbf.WriteCloser) {
	// utils.DebugLog("send HeartBeat")
	msg := msg.RelayMsgBuf{
		MSGHead: PPMsgHeader(nil, header.RspHeart),
	}
	conn.Write(&msg)
}

// RspHeartBeat
func RspHeartBeat(ctx context.Context, conn spbf.WriteCloser) {
	// utils.DebugLog("ResHeartBeat")
	switch conn.(type) {
	case *spbf.ServerConn:
		msg := msg.RelayMsgBuf{
			MSGHead: PPMsgHeader(nil, header.RspHeart),
		}
		conn.Write(&msg)
	}
}
