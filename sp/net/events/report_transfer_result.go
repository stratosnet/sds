package events

import (
	"context"
	"github.com/qsnetwork/qsds/framework/spbf"
	"github.com/qsnetwork/qsds/msg/header"
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/sp/net"
	"github.com/qsnetwork/qsds/sp/storages/table"
	"github.com/qsnetwork/qsds/utils"
	"time"
)

type ReportTransferResult struct {
	Server *net.Server
}

// GetServer
func (e *ReportTransferResult) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *ReportTransferResult) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *ReportTransferResult) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqReportTransferResult)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqReportTransferResult)

		rsp := &protos.RspReportTransferResult{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			TransferCer: body.TransferCer,
		}

		if body.Result.State != protos.ResultState_RES_SUCCESS ||
			body.TransferCer == "" {

			rsp.Result.Msg = "report result failed"
			rsp.Result.State = protos.ResultState_RES_FAIL

			return rsp, header.RspReportTransferResult
		}

		// todo change to read from redis
		transferRecord := new(table.TransferRecord)
		transferRecord.TransferCer = body.TransferCer

		e.GetServer().Lock()
		if e.GetServer().Load(transferRecord) != nil {
			rsp.Result.Msg = "transfer record doesn't exist"
			rsp.Result.State = protos.ResultState_RES_FAIL
			e.GetServer().Unlock()
			return rsp, header.RspReportTransferResult
		}

		if transferRecord.SliceHash == "" {

			rsp.Result.Msg = "transfer record info error, empty slice hash"
			rsp.Result.State = protos.ResultState_RES_FAIL
			e.GetServer().Unlock()
			return rsp, header.RspReportTransferResult
		}

		if transferRecord.Status == table.TRANSFER_RECORD_STATUS_CHECK {
			transferRecord.Status = table.TRANSFER_RECORD_STATUS_CONFIRM
		} else if transferRecord.Status == table.TRANSFER_RECORD_STATUS_CONFIRM {
			transferRecord.Status = table.TRANSFER_RECORD_STATUS_SUCCESS
		} else {

			e.GetServer().Unlock()
			return rsp, header.RspReportTransferResult
		}

		transferRecord.Time = time.Now().Unix()

		// todo change to read from redis
		e.GetServer().Store(transferRecord, 3600*time.Second)
		e.GetServer().Unlock()

		if transferRecord.Status == table.TRANSFER_RECORD_STATUS_SUCCESS {

			//todo change to read from redis
			if ok, err := e.GetServer().CT.StoreTable(transferRecord); !ok {

				utils.ErrorLog(err)
			}

			if body.IsNew {
				// todo new pp's transfer report, need to process
			}

			fileSlice := new(table.FileSlice)
			fileSlice.SliceHash = transferRecord.SliceHash
			fileSlice.WalletAddress = transferRecord.FromWalletAddress

			if e.GetServer().CT.Fetch(fileSlice) == nil {

				fileSliceStorage := &table.FileSliceStorage{
					SliceHash:      fileSlice.SliceHash,
					WalletAddress:  body.NewPp.WalletAddress,
					NetworkAddress: body.NewPp.NetworkAddress,
				}
				e.GetServer().CT.StoreTable(fileSliceStorage)
			}

			// todo change to read from redis
			e.GetServer().Remove(transferRecord.GetCacheKey())
		}

		return rsp, header.RspReportTransferResult
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
