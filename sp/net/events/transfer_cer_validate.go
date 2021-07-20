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

// transferCerValidate is a concrete implementation of event
type transferCerValidate struct {
	event
}

const transferCerValidateEvent = "transfer_cer_validate"

// GetTransferCerValidateHandler creates event and return handler func for it
func GetTransferCerValidateHandler(s *net.Server) EventHandleFunc {
	e := transferCerValidate{newEvent(transferCerValidateEvent, s, transferCerValidateCallbackFunc)}
	return e.Handle
}

// transferCerValidateCallbackFunc is the main process of transfer cer validate
func transferCerValidateCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqValidateTransferCer)

	rsp := &protos.RspValidateTransferCer{
		TransferCer: body.TransferCer,
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
	}

	if body.TransferCer == "" {
		rsp.Result.Msg = "transfer certificate can't be empty"
		rsp.Result.State = protos.ResultState_RES_FAIL
		return rsp, header.RspValidateTransferCer
	}

	// todo change to read from redis
	transferRecord := &table.TransferRecord{
		TransferCer: body.TransferCer,
	}

	if s.Load(transferRecord) != nil {
		rsp.Result.Msg = "failed to validate transfer certificate"
		rsp.Result.State = protos.ResultState_RES_FAIL
		return rsp, header.RspValidateTransferCer
	}

	if transferRecord.ToP2pAddress == "" || transferRecord.Status != table.TRANSFER_RECORD_STATUS_CHECK {
		rsp.Result.Msg = "transfer certificate invalid, empty destination"
		rsp.Result.State = protos.ResultState_RES_FAIL
		return rsp, header.RspValidateTransferCer
	}

	if transferRecord.ToP2pAddress != body.NewPp.P2PAddress {
		rsp.Result.Msg = "transfer certificate invalid, P2P key addresses don't match"
		rsp.Result.State = protos.ResultState_RES_FAIL
		return rsp, header.RspValidateTransferCer
	}

	fileSlice := &table.FileSlice{
		SliceHash: transferRecord.SliceHash,
		FileSliceStorage: table.FileSliceStorage{
			P2pAddress: transferRecord.FromP2pAddress,
		},
	}

	if s.CT.Fetch(fileSlice) != nil || fileSlice.P2pAddress != body.OriginalPp.P2PAddress {
		rsp.Result.Msg = "file slice not exist or the original PP doesn't have it"
		rsp.Result.State = protos.ResultState_RES_FAIL
		return rsp, header.RspValidateTransferCer
	}

	transferRecord.ToP2pAddress = body.NewPp.P2PAddress
	transferRecord.ToWalletAddress = body.NewPp.WalletAddress
	transferRecord.ToNetworkAddress = body.NewPp.NetworkAddress
	transferRecord.Time = time.Now().Unix()

	utils.Log("created transfer certificate：Slice: " + fileSlice.SliceHash + " From[" + fileSlice.P2pAddress + "] to[" + body.NewPp.P2PAddress + "]")

	// todo change to read from redis
	if err := s.Store(transferRecord, 3600*time.Second); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, transferNoticeEvent, "store transfer record to db", err)
	}

	if rsp.Result.State == protos.ResultState_RES_FAIL {
		// todo prepare another transfer certificate
	}

	return rsp, header.RspValidateTransferCer
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *transferCerValidate) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqValidateTransferCer{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
