package network

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/setting"
)

// StartPpLatencyCheck
func (p *Network) StartPpLatencyCheck(ctx context.Context) {
	p.ppPeerClock.AddJobRepeat(time.Second*setting.PpLatencyCheckInterval, 0, p.LatencyOfNextPp(ctx))
}

//LantencyOfNextPp
func (p *Network) LatencyOfNextPp(ctx context.Context) func() {
	return func() {
		list, _, _ := p2pserver.GetP2pServer(ctx).GetPPList(ctx)
		for _, peer := range list {
			if peer.Latency == 0 {
				p.StartLatencyCheckToPp(ctx, peer.NetworkAddress)
			}
		}
	}
}

// StartLatencyCheckToPp
func (p *Network) StartLatencyCheckToPp(ctx context.Context, NetworkAddr string) error {
	start := time.Now().UnixNano()
	p.pingTimePPMap.Store(NetworkAddr, start)
	pb := &protos.ReqLatencyCheck{
		HbType: protos.HeartbeatType_LATENCY_CHECK_PP,
	}
	data, err := proto.Marshal(pb)
	if err != nil {
		return err
	}

	msg := &msg.RelayMsgBuf{
		MSGHead: header.MakeMessageHeader(1, uint16(setting.Config.Version.AppVer), uint32(len(data)), header.ReqLatencyCheck),
		MSGData: data,
	}

	p2pserver.GetP2pServer(ctx).TransferSendMessageToPPServ(ctx, NetworkAddr, msg)
	return nil
}
