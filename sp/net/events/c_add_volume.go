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
	"time"
)

// cAddVolume is a concrete implementation of event
// customer purchase volume from Stratos, then RelayD publish this event to SP
type cAddVolume struct {
	event
}

const cAddVolumeEvent = "customer_add_volume"

// GetCAddVolumeHandler creates event and return handler func for it
func GetCAddVolumeHandler(s *net.Server) EventHandleFunc {
	e := cAddVolume{newEvent(cAddVolumeEvent, s, getCustomerAddVolumeCallbackFunc)}
	return e.Handle
}

// getCustomerAddVolumeCallbackFunc is the main process of customer add volume
func getCustomerAddVolumeCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqCustomerAddVolume)

	rsp := &protos.RspCustomerAddVolume{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		WalletAddress: body.PpBaseInfo.WalletAddress,
		ReqId:         body.ReqId,
	}

	if body.PpBaseInfo.WalletAddress == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "wallet address can't be empty"
		return rsp, header.RspCAddVolume
	}

	customer := &table.Customer{WalletAddress: body.PpBaseInfo.GetWalletAddress()}

	if err := s.CT.Fetch(customer); err != nil {
		// new customer
		customer.RegisterTime = time.Now().Unix()
	}

	customer.WalletAddress = body.PpBaseInfo.WalletAddress
	customer.NetworkAddress = body.PpBaseInfo.NetworkId.NetworkAddress
	customer.Puk = body.PpBaseInfo.NetworkId.PublicKey
	customer.LastLoginTime = time.Now().Unix()
	customer.LoginTimes++
	customer.TotalVolume += body.Volume

	if err := s.CT.Save(customer); err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "save new customer failed"
		return rsp, header.RspCAddVolume
	}

	return rsp, header.RspCAddVolume
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e cAddVolume) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqCustomerAddVolume{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
