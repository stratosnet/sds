package events

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
)

// reportTransferResult is a concrete implementation of event
type reportTransferResult struct {
	event
}

const reportTransferResultEvent = "report_transfer_result"

// GetReportTransferResultHandler creates event and return handler func for it
func GetReportTransferResultHandler(s *net.Server) EventHandleFunc {
	e := reportTransferResult{newEvent(reportTransferResultEvent, s, reportTransferResultCallbackFunc)}
	return e.Handle
}

func reportTransferResultCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
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
	transferRecord := &table.TransferRecord{
		TransferCer: body.TransferCer,
	}

	s.Lock()
	if s.Load(transferRecord) != nil {
		rsp.Result.Msg = "transfer record doesn't exist"
		rsp.Result.State = protos.ResultState_RES_FAIL
		s.Unlock()
		return rsp, header.RspReportTransferResult
	}

	if transferRecord.SliceHash == "" {

		rsp.Result.Msg = "transfer record info error, empty slice hash"
		rsp.Result.State = protos.ResultState_RES_FAIL
		s.Unlock()
		return rsp, header.RspReportTransferResult
	}

	if transferRecord.Status == table.TRANSFER_RECORD_STATUS_CHECK {
		transferRecord.Status = table.TRANSFER_RECORD_STATUS_CONFIRM
	} else if transferRecord.Status == table.TRANSFER_RECORD_STATUS_CONFIRM {
		transferRecord.Status = table.TRANSFER_RECORD_STATUS_SUCCESS
	} else {

		s.Unlock()
		return rsp, header.RspReportTransferResult
	}

	transferRecord.Time = time.Now().Unix()

	// todo change to read from redis
	if err := s.Store(transferRecord, 3600*time.Second); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, reportTransferResultEvent, "store transfer report to db", err)
	}
	s.Unlock()

	if transferRecord.Status != table.TRANSFER_RECORD_STATUS_SUCCESS {
		return rsp, header.RspReportTransferResult
	}

	//todo change to read from redis
	if ok, err := s.CT.StoreTable(transferRecord); !ok {
		utils.ErrorLogf(eventHandleErrorTemplate, reportTransferResultEvent, "store transfer record table to db", err)
	}

	if body.IsNew {
		// todo new pp's transfer report, need to process
	}

	fileSlice := &table.FileSlice{
		SliceHash: transferRecord.SliceHash,
		FileSliceStorage: table.FileSliceStorage{
			WalletAddress: transferRecord.FromWalletAddress,
		},
	}

	if err := s.CT.Fetch(fileSlice); err == nil {

		fileSliceStorage := &table.FileSliceStorage{
			SliceHash:      fileSlice.SliceHash,
			WalletAddress:  body.NewPp.P2PAddress,
			NetworkAddress: body.NewPp.NetworkAddress,
		}

		if _, err = s.CT.StoreTable(fileSliceStorage); err != nil {
			utils.ErrorLogf(eventHandleErrorTemplate, reportTransferResultEvent, "store file slice storage table to db", err)
		}
	}

	// todo change to read from redis
	if err := s.Remove(transferRecord.GetCacheKey()); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, reportTransferResultEvent, "remove transfer record from db", err)
	}

	trafficRecord := &table.Traffic{
		TaskId:                body.TransferCer,
		TaskType:              table.TRAFFIC_TASK_TYPE_TRANSFER,
		ProviderWalletAddress: transferRecord.FromWalletAddress,
		ConsumerWalletAddress: transferRecord.ToWalletAddress,
		Volume:                transferRecord.SliceSize,
		DeliveryTime:          transferRecord.Time,
	}

	if ok, err := s.CT.StoreTable(trafficRecord); !ok {
		utils.ErrorLogf(eventHandleErrorTemplate, reportTransferResultEvent, "store traffic record table to db", err)
	}

	return rsp, header.RspReportTransferResult
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *reportTransferResult) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqReportTransferResult{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()

}
