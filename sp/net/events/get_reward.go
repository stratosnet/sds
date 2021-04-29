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

// getReward is a concrete implementation of event
type getReward struct {
	event
}

const getRewardEvent = "get_reward"

// GetGetRewardHandler creates event and return handler func for it
func GetGetRewardHandler(s *net.Server) EventHandleFunc {
	e := getReward{newEvent(getRewardEvent, s, getRewardCallbackFunc)}
	return e.Handle
}

// getRewardCallbackFunc is the main process of get rewarding
func getRewardCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqGetReward)
	rsp := &protos.RspGetReward{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		ReqId:         body.ReqId,
		WalletAddress: body.WalletAddress,
	}

	if body.WalletAddress == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "wallet address can't be empty"
		return rsp, header.RspGetReward
	}

	user := &table.User{
		WalletAddress: body.WalletAddress,
	}

	if err := s.CT.Fetch(user); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "need to login first"
		return rsp, header.RspGetReward
	}

	invite := &table.UserInvite{
		InvitationCode: user.InvitationCode,
	}

	if err := s.CT.Fetch(invite); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "invitation code is invalid"
		return rsp, header.RspGetReward
	}

	if invite.Times < 5 {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "invite times not enough(need 5 times)"
		return rsp, header.RspGetReward
	}

	if user.IsUpgrade == 1 {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "already upgraded"
		return rsp, header.RspGetReward
	}

	user.Capacity = user.Capacity + s.System.UpgradeReward
	user.IsUpgrade = 1

	if err := s.CT.Save(user); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = err.Error()
		return rsp, header.RspGetReward
	}

	rsp.CurrentCapacity = user.GetCapacity()

	return rsp, header.RspGetReward
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *getReward) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqGetReward{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
