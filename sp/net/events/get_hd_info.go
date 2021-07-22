package events

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
)

type getHDInfo struct {
	event
}

const getHdInfoEvent = "get_hd_info"

func GetHDInfoHandler(s *net.Server) EventHandleFunc {
	e := getHDInfo{newEvent(getHdInfoEvent, s, getHdInfoCallbackFunc)}
	return e.Handle
}

func getHdInfoCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.RspGetHDInfo)

	user := &table.User{P2pAddress: body.P2PAddress}

	if s.CT.Fetch(user) != nil {
		return nil, ""
	}

	user.FreeDisk = body.DiskFree
	user.DiskSize = body.DiskSize
	if err := s.CT.Save(user); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, getHdInfoEvent, "save user to db", err)
	}

	if user.IsPp != 1 {
		return nil, ""
	}

	pp := &table.PP{P2pAddress: body.P2PAddress}

	if s.CT.Fetch(pp) == nil {
		return nil, ""
	}

	pp.DiskSize = body.DiskSize
	pp.FreeDisk = body.DiskFree
	if err := s.CT.Save(pp); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, getHdInfoEvent, "save pp to db", err)
	}

	// check if reported disk size is within configured scale
	if float32(body.DiskSize-body.DiskFree)/float32(body.DiskSize) > s.Conf.Peers.ProvideDiskScale {

		// if over sized, removed from hashring, don't assign task anymore
		s.HashRing.SetOffline(body.P2PAddress)

		// todo: disk clean or file re-allocation?
	}

	return nil, ""
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *getHDInfo) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.RspGetHDInfo{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
