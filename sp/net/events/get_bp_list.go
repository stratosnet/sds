package events

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/utils"
)

// getBPList is a concrete implementation of event
type getBPList struct {
	event
}

const getBpListEvent = "get_bp_list"

// GetBPListHandler creates event and return handler func for it
func GetBPListHandler(s *net.Server) EventHandleFunc {
	return getBPList{newEvent(getBpListEvent, s, getBpListCallbackFunc)}.Handle
}

// getBpListCallbackFunc is the main process of get bp list
func getBpListCallbackFunc(_ context.Context, s *net.Server, _ proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	bps := s.Conf.BpList

	bpList := make([]*protos.PPBaseInfo, 0, len(bps))

	for i := 0; i < len(bps); i++ {
		info := &protos.PPBaseInfo{
			NetworkAddress: bps[i].NetworkAddress,
			WalletAddress:  bps[i].WalletAddress,
		}
		bpList = append(bpList, info)
	}

	rsp := &protos.RspGetBPList{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		BpList: bpList,
	}

	return rsp, header.RspGetBPList
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *getBPList) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := new(protos.ReqGetBPList)
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
