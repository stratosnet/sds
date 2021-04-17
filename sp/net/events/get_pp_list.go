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

// GetPPList is a concrete implementation of event
type GetPPList struct {
	event
}

// GetPPListHandler creates event and return handler func for it
func GetPPListHandler(s *net.Server) EventHandleFunc {
	return GetPPList{
		newEvent("get_pp_list", s, getPPListCallbackFunc),
	}.Handle
}

// getPPListCallbackFunc is the main process of get pp list
func getPPListCallbackFunc(_ context.Context, s *net.Server, _ proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	// get PP from hash ring
	ppList := s.HashRing.RandomGetNodes(s.Conf.Peers.List)

	ppBaseInfoList := make([]*protos.PPBaseInfo, 0, len(ppList))

	for _, pp := range ppList {
		ppBaseInfo := &protos.PPBaseInfo{
			WalletAddress:  pp.ID,
			NetworkAddress: pp.Host,
		}
		ppBaseInfoList = append(ppBaseInfoList, ppBaseInfo)
	}

	rsp := &protos.RspGetPPList{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		PpList: ppBaseInfoList,
	}

	return rsp, header.RspGetPPList
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *GetPPList) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		if err := e.handle(ctx, conn, &protos.ReqGetPPList{}); err != nil {
			utils.ErrorLog(err)
		}
	}()

}
