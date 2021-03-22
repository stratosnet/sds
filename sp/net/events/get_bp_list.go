package events

import (
	"context"
	"github.com/qsnetwork/qsds/framework/spbf"
	"github.com/qsnetwork/qsds/msg/header"
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/sp/net"
)

// GetBPList
type GetBPList struct {
	Server *net.Server
}

// GetServer
func (e *GetBPList) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *GetBPList) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *GetBPList) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqGetBPList)

	callback := func(message interface{}) (interface{}, string) {

		bps := e.GetServer().Conf.BpList

		bpList := make([]*protos.PPBaseInfo, 0, len(bps))

		if len(bps) > 0 {
			for i := 0; i < len(bps); i++ {
				info := &protos.PPBaseInfo{
					NetworkAddress: bps[i].NetworkAddress,
					WalletAddress:  bps[i].WalletAddress,
				}
				bpList = append(bpList, info)
			}
		}

		rsp := &protos.RspGetBPList{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			BpList: bpList,
		}

		return rsp, header.RspGetBPList
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
