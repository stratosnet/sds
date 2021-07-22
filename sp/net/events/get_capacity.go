package events

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
)

// getCapacity is a concrete implementation of event
type getCapacity struct {
	event
}

const getCapacityEvent = "get_capacity"

// GetGetCapacityHandler creates event and return handler func for it
func GetGetCapacityHandler(s *net.Server) EventHandleFunc {
	e := getCapacity{newEvent(getCapacityEvent, s, getCapacityCallbackFunc)}
	return e.Handle
}

func getCapacityCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqGetCapacity)

	rsp := &protos.RspGetCapacity{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		P2PAddress:    body.P2PAddress,
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
		Capacity:      0,
		FreeCapacity:  0,
	}

	if body.P2PAddress == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "P2P key address can't be empty"
		return rsp, header.RspGetCapacity
	}

	user := &table.User{P2pAddress: body.P2PAddress}
	if err := s.CT.Fetch(user); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "need to login wallet first"
		return rsp, header.RspConfig
	}

	totalUsed, err := s.CT.SumTable(&table.File{}, "f.size", map[string]interface{}{
		"alias": "f",
		"join":  []string{"user_has_file", "f.hash = uhf.file_hash", "uhf"},
		"where": map[string]interface{}{"uhf.wallet_address = ?": user.WalletAddress},
	})

	if err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, getCapacityEvent, "sum file table from db", err)
	}

	user.UsedCapacity = uint64(totalUsed)
	if err = s.CT.Update(user); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, getCapacityEvent, "update user in db", err)
	}

	rsp.Capacity = user.GetCapacity()
	rsp.FreeCapacity = user.GetFreeCapacity()

	return rsp, header.RspGetCapacity
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *getCapacity) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqGetCapacity{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
