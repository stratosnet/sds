package events

import (
	"context"
	"encoding/hex"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
	"time"
)

// cUseVolume is a concrete implementation of event
// customer uses volume from Stratos, then RelayD publish this event to SP
type cUseVolume struct {
	event
}

const cUseVolumeEvent = "customer_use_volume"

// GetCUseVolumeHandler creates event and return handler func for it
func GetCUseVolumeHandler(s *net.Server) EventHandleFunc {
	e := cAddVolume{newEvent(cAddVolumeEvent, s, getCustomerUseVolumeCallbackFunc)}
	return e.Handle
}

// getCustomerUseVolumeCallbackFunc is the main process of customer expense volume
func getCustomerUseVolumeCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqCustomerUseVolume)

	rsp := &protos.RspCustomerUseVolume{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
	}

	if body.WalletAddress == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "wallet address can't be empty"
		return rsp, header.RspCUseVolume
	}

	customer := &table.Customer{WalletAddress: body.WalletAddress}

	if err := s.CT.Fetch(customer); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "customer not found"
		return rsp, header.RspCUseVolume
	}

	if customer.GetAvailableVolume() < body.RequiredVolume {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "insufficient volume"
		return rsp, header.RspCUseVolume
	}

	customer.WalletAddress = body.WalletAddress
	customer.Puk = hex.EncodeToString(body.PublicKey)
	customer.LastLoginTime = time.Now().Unix()
	customer.LoginTimes++
	customer.UsedVolume += body.RequiredVolume

	if err := s.CT.Save(customer); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "save new customer failed"
		return rsp, header.RspCAddVolume
	}

	return rsp, header.RspCAddVolume
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e cUseVolume) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqCustomerUseVolume{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
