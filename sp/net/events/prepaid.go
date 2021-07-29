package events

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
	"math/big"
)

// prepaid is a concrete implementation of event
// stratoschain prepay transaction success
type prepaid struct {
	event
}

const prepaidEvent = "prepaid"

// GetPrepaidHandler creates event and return handler func for it
func GetPrepaidHandler(s *net.Server) EventHandleFunc {
	e := prepaid{newEvent(prepaidEvent, s, prepaidCallbackFunc)}
	return e.Handle
}

// prepaidCallbackFunc is the main process of updating the user capacity following a successful prepay transaction
func prepaidCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqPrepaid)

	rsp := &protos.RspPrepaid{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
	}

	userOzone := &table.UserOzone{
		WalletAddress: body.WalletAddress,
		AvailableUoz:  big.NewInt(0).String(),
	}
	err := s.CT.Fetch(userOzone)
	if err != nil {
		err = s.CT.Save(userOzone)
		if err != nil {
			utils.ErrorLog("Couldn't save user ozone to database")
			return rsp, header.RspPrepaid
		}
	}

	availableUoz := &big.Int{}
	_, success := availableUoz.SetString(userOzone.AvailableUoz, 10)
	if !success {
		utils.ErrorLog(fmt.Sprintf("Invalid user ozone stored in database: {%v}. User ozone set to 0.", userOzone.AvailableUoz))
		_ = availableUoz.Set(big.NewInt(0))
	}

	purchasedUoz := &big.Int{}
	_, success = purchasedUoz.SetString(body.PurchasedUoz, 10)
	if !success {
		utils.ErrorLog("Invalid purchased ozone in ReqPrepaid message: " + body.PurchasedUoz)
		return rsp, header.RspPrepaid
	}

	_ = availableUoz.Add(availableUoz, purchasedUoz)
	userOzone.AvailableUoz = availableUoz.String()
	if err := s.CT.Update(userOzone); err != nil {
		utils.ErrorLog(err)
	}
	return rsp, header.RspPrepaid
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *prepaid) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqPrepaid{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
