package network

import (
	"context"
	"time"

	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/setting"
	"google.golang.org/protobuf/proto"
)

func (p *Network) SchedulePpLatencyCheck(ctx context.Context) {
	p.ppPeerClock.AddJobRepeat(time.Second*setting.PpLatencyCheckInterval, 0, p.LatencyOfNextPp(ctx))
}

// LatencyOfNextPp only when the Latency is 0 which means not measured, the measurement is started.
func (p *Network) LatencyOfNextPp(ctx context.Context) func() {
	return func() {
		list, _, _ := p2pserver.GetP2pServer(ctx).GetPPList(ctx)
		for _, peer := range list {
			if peer.Latency == 0 {
				_ = p.StartLatencyCheckToPp(ctx, peer.NetworkAddress)
			}
		}
	}
}

// StartLatencyCheckToPp send pp latency check message to measure the latency
func (p *Network) StartLatencyCheckToPp(ctx context.Context, NetworkAddr string) error {
	start := time.Now().UnixNano()
	p.pingTimePPMap.Store(NetworkAddr, start)
	pb := &protos.ReqPpLatencyCheck{}
	body, err := proto.Marshal(pb)
	if err != nil {
		return err
	}

	msgRelay := &msg.RelayMsgBuf{
		MSGHead: header.MakeMessageHeader(1, setting.Config.Version.AppVer, uint32(len(body)), header.ReqPpLatencyCheck),
		MSGBody: body,
	}

	return p2pserver.GetP2pServer(ctx).TransferSendMessageToPPServ(ctx, NetworkAddr, msgRelay)
}
