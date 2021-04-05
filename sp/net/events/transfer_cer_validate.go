package events

import (
	"context"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg/header"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/sp/net"
	"github.com/qsnetwork/sds/sp/storages/table"
	"github.com/qsnetwork/sds/utils"
	"time"
)

// TransferCerValidate
type TransferCerValidate struct {
	Server *net.Server
}

// GetServer
func (e *TransferCerValidate) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *TransferCerValidate) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *TransferCerValidate) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqValidateTransferCer)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqValidateTransferCer)

		rsp := new(protos.RspValidateTransferCer)
		rsp.TransferCer = body.TransferCer
		rsp.Result = &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		}

		if body.TransferCer == "" {
			rsp.Result.Msg = "transfer certificate can't be empty"
			rsp.Result.State = protos.ResultState_RES_FAIL
			return rsp, header.RspValidateTransferCer
		}

		// todo change to read from redis
		transferRecord := new(table.TransferRecord)

		transferRecord.TransferCer = body.TransferCer

		if e.GetServer().Load(transferRecord) != nil {

			rsp.Result.Msg = "failed to validate transfer certificate"
			rsp.Result.State = protos.ResultState_RES_FAIL
			return rsp, header.RspValidateTransferCer
		}


		if transferRecord.ToWalletAddress == "" ||
			transferRecord.Status != table.TRANSFER_RECORD_STATUS_CHECK {

			rsp.Result.Msg = "transfer certificate invalid, empty destination"
			rsp.Result.State = protos.ResultState_RES_FAIL
			return rsp, header.RspValidateTransferCer
		}


		if transferRecord.ToWalletAddress != body.NewPp.WalletAddress {

			rsp.Result.Msg = "transfer certificate invalid, wallet address not match"
			rsp.Result.State = protos.ResultState_RES_FAIL
			return rsp, header.RspValidateTransferCer
		}

		fileSlice := new(table.FileSlice)
		fileSlice.SliceHash = transferRecord.SliceHash
		fileSlice.WalletAddress = transferRecord.FromWalletAddress

		if e.GetServer().CT.Fetch(fileSlice) != nil ||
			fileSlice.WalletAddress != body.OriginalPp.WalletAddress {

			rsp.Result.Msg = "file slice not exist or the original PP doesn't have it"
			rsp.Result.State = protos.ResultState_RES_FAIL
			return rsp, header.RspValidateTransferCer
		}

		transferRecord.ToNetworkAddress = body.NewPp.NetworkAddress
		transferRecord.ToWalletAddress = body.NewPp.WalletAddress
		transferRecord.Time = time.Now().Unix()

		utils.Log("created transfer certificate：Slice: " + fileSlice.SliceHash + " From[" + fileSlice.WalletAddress + "] to[" + body.NewPp.WalletAddress + "]")

		// todo change to read from redis
		e.GetServer().Store(transferRecord, 3600*time.Second)

		if rsp.Result.State == protos.ResultState_RES_FAIL {
			// todo prepare another transfer certificate
		}

		return rsp, header.RspValidateTransferCer
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
