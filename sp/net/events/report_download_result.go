package events

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/data"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
)

// reportDownloadResult is a concrete implementation of event
type reportDownloadResult struct {
	event
}

const reportDownloadResultEvent = "report_download_event"

// GetReportDownloadResultHandler creates event and return handler func for it
func GetReportDownloadResultHandler(s *net.Server) EventHandleFunc {
	e := reportDownloadResult{newEvent(reportDownloadResultEvent, s, reportDownloadResultCallbackFunc)}
	return e.Handle
}

// reportDownloadResultCallbackFunc is the main process of report download result
func reportDownloadResultCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqReportDownloadResult)

	rsp := &protos.RspReportDownloadResult{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		SliceInfo: body.SliceInfo,
	}

	if ok, msg := validateReportDownloadRequest(body); !ok {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = msg
		return rsp, header.RspReportDownloadResult
	}

	fileHash := body.FileHash
	sliceHash := body.SliceInfo.SliceStorageInfo.SliceHash

	fileSlice := &table.FileSlice{}

	// query reported file or slice exist or not todo change to read from redis
	err := s.CT.FetchTable(fileSlice, map[string]interface{}{
		"where": map[string]interface{}{
			"file_hash = ? AND slice_hash = ?": []interface{}{
				fileHash, sliceHash,
			},
		},
	})

	if err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "slice not exist, report error"
		return rsp, header.RspReportDownloadResult
	}

	record := &table.FileSliceDownload{TaskId: body.TaskId}

	s.Lock()
	if s.Load(record) == nil {
		if record.Status == table.DOWNLOAD_STATUS_SUCCESS {
			s.Unlock()
			return rsp, header.RspReportDownloadResult
		}

		record.Status = table.DOWNLOAD_STATUS_SUCCESS
		record.Time = time.Now().Unix()

		if record.SliceHash != fileSlice.SliceHash {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "report validation error"
			s.Unlock()
			return rsp, header.RspReportDownloadResult
		}
	} else {
		record.SliceHash = fileSlice.SliceHash
		record.Status = table.DOWNLOAD_STATUS_CHECK
		record.Time = time.Now().Unix()
	}

	// TODO: confirm this logic in QB-475
	if body.IsPP {
		record.FromP2PAddress = body.MyP2PAddress
		record.FromWalletAddress = body.MyWalletAddress
		record.ToP2PAddress = body.DownloaderP2PAddress
		record.ToWalletAddress = body.DownloaderWalletAddress
	} else {
		record.FromP2PAddress = body.DownloaderP2PAddress
		record.FromWalletAddress = body.DownloaderWalletAddress
		record.ToP2PAddress = body.MyP2PAddress
		record.ToWalletAddress = body.MyWalletAddress
	}

	if err = s.Store(record, 3600*time.Second); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, reportDownloadResultEvent, "store record to cache", err)
	}

	s.Unlock()

	if record.Status == table.DOWNLOAD_STATUS_SUCCESS {

		if ok, err := s.CT.StoreTable(record); !ok {
			utils.ErrorLogf(eventHandleErrorTemplate, reportDownloadResultEvent, "store record table to db", err)
		}

		//persist Traffic records
		traffic := &table.Traffic{
			TaskId:                body.TaskId,
			ProviderP2PAddress:    record.FromP2PAddress,
			ProviderWalletAddress: record.FromWalletAddress,
			ConsumerWalletAddress: record.ToWalletAddress,
			TaskType:              table.TRAFFIC_TASK_TYPE_DOWNLOAD,
			Volume:                fileSlice.SliceSize,
			DeliveryTime:          time.Now().Unix(),
		}

		if ok, err := s.CT.StoreTable(traffic); !ok {
			utils.ErrorLogf(eventHandleErrorTemplate, reportDownloadResultEvent, "store traffic record table to db", err)
		}

		// verify download
		downloadFile := &data.DownloadFile{FileHash: fileSlice.FileHash, WalletAddress: body.DownloaderWalletAddress}
		if s.Load(downloadFile) == nil {
			downloadFile.SetSliceFinish(fileSlice.SliceNumber)
			if err = s.Store(downloadFile, 7*24*3600*time.Second); err != nil {
				utils.ErrorLogf(eventHandleErrorTemplate, reportDownloadResultEvent, "store download file to db", err)
			}
			if downloadFile.IsDownloadFinished() {
				fileDownload := &table.FileDownload{
					FileHash:        fileSlice.FileHash,
					ToWalletAddress: body.DownloaderWalletAddress,
					TaskId:          utils.CalcHash([]byte(fileSlice.FileHash + body.DownloaderWalletAddress)),
					Time:            time.Now().Unix(),
				}
				if _, err = s.CT.StoreTable(fileDownload); err != nil {
					utils.ErrorLogf(eventHandleErrorTemplate, reportDownloadResultEvent, "store file download table to db", err)
				}
			}
		}

		if err = s.Remove(record.GetCacheKey()); err != nil {
			utils.ErrorLogf(eventHandleErrorTemplate, reportDownloadResultEvent, "remove record from db", err)
		}
	}

	return rsp, header.RspReportDownloadResult
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *reportDownloadResult) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqReportDownloadResult{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}

// validateReportDownloadRequest validates request
func validateReportDownloadRequest(req *protos.ReqReportDownloadResult) (bool, string) {

	if req.SliceInfo == nil ||
		req.SliceInfo.SliceStorageInfo == nil ||
		req.FileHash == "" ||
		req.SliceInfo.SliceStorageInfo.SliceHash == "" {

		return false, "report info invalid"
	}

	if req.TaskId == "" {
		return false, "download task ID can't be empty"
	}

	//user := &table.User{WalletAddress: req.WalletAddress}
	//if s.CT.Fetch(user) != nil {
	//	return false, "not authorized to process"
	//}
	//
	//pukInByte, err := hex.DecodeString(user.Puk)
	//if err != nil {
	//	return false, err.Error()
	//}
	//
	//data := req.MyP2PAddress + req.FileHash
	//if !ed25519.Verify(puk, []byte(data), req.Sign) {
	//	return false, "signature verification failed"
	//}

	return true, ""
}
