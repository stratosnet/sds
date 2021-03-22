package events

import (
	"context"
	"github.com/qsnetwork/qsds/framework/spbf"
	"github.com/qsnetwork/qsds/msg/header"
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/sp/net"
)

// GetPPList
type GetPPList struct {
	Server *net.Server
}

// GetServer
func (e *GetPPList) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *GetPPList) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *GetPPList) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqGetPPList)

	callback := func(message interface{}) (interface{}, string) {

		// get PP from hashring
		ppList := e.GetServer().HashRing.RandomGetNodes(e.GetServer().Conf.Peers.List)

		ppBaseInfoList := make([]*protos.PPBaseInfo, 0, len(ppList))

		if len(ppList) > 0 {

			for _, pp := range ppList {
				ppBaseInfo := &protos.PPBaseInfo{
					WalletAddress:  pp.ID,
					NetworkAddress: pp.Host,
				}
				ppBaseInfoList = append(ppBaseInfoList, ppBaseInfo)
			}
		}

		rsp := &protos.RspGetPPList{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			PpList: ppBaseInfoList,
		}

		return rsp, header.RspGetPPList
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
