package events

import (
	"context"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/data"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
	"time"
)

// ReportDownloadResult
type ReportDownloadResult struct {
	Server *net.Server
}

// GetServer
func (e *ReportDownloadResult) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *ReportDownloadResult) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *ReportDownloadResult) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqReportDownloadResult)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqReportDownloadResult)

		rsp := &protos.RspReportDownloadResult{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			SliceInfo: body.SliceInfo,
		}

		if ok, msg := e.Validate(body); !ok {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = msg
			return rsp, header.RspReportDownloadResult
		}

		fileHash := body.FileHash
		sliceHash := body.SliceInfo.SliceStorageInfo.SliceHash

		fileSlice := new(table.FileSlice)

		// query reported file or slice exist or not todo change to read from redis
		err := e.GetServer().CT.FetchTable(fileSlice, map[string]interface{}{
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

		record := &table.FileSliceDownload{
			TaskId: body.TaskId,
		}

		e.GetServer().Lock()
		if e.GetServer().Load(record) == nil {

			if record.Status == table.DOWNLOAD_STATUS_SUCCESS {
				e.GetServer().Unlock()
				return rsp, header.RspReportDownloadResult
			}

			record.Status = table.DOWNLOAD_STATUS_SUCCESS
			record.Time = time.Now().Unix()

			if record.SliceHash != fileSlice.SliceHash {
				rsp.Result.State = protos.ResultState_RES_FAIL
				rsp.Result.Msg = "report validation error"
				e.GetServer().Unlock()
				return rsp, header.RspReportDownloadResult
			}

		} else {


			record.SliceHash = fileSlice.SliceHash
			record.Status = table.DOWNLOAD_STATUS_CHECK
			record.Time = time.Now().Unix()
		}

		if body.IsPP {
			record.FromWalletAddress = body.WalletAddress
		} else {
			record.ToWalletAddress = body.WalletAddress
		}

		e.GetServer().Store(record, 3600*time.Second)

		e.GetServer().Unlock()

		if record.Status == table.DOWNLOAD_STATUS_SUCCESS {

			if ok, err := e.GetServer().CT.StoreTable(record); !ok {
				utils.ErrorLog(err.Error())
			}

			// 下载校对
			downloadFile := &data.DownloadFile{FileHash: fileSlice.FileHash, WalletAddress: body.WalletAddress}
			if e.GetServer().Load(downloadFile) == nil {
				downloadFile.SetSliceFinish(fileSlice.SliceNumber)
				e.GetServer().Store(downloadFile, 7*24*3600*time.Second)
				if downloadFile.IsDownloadFinished() {
					fileDownload := &table.FileDownload{
						FileHash:        fileSlice.FileHash,
						ToWalletAddress: body.WalletAddress,
						TaskId:          utils.CalcHash([]byte(fileSlice.FileHash + body.WalletAddress)),
						Time:            time.Now().Unix(),
					}
					e.GetServer().CT.StoreTable(fileDownload)
				}
			}

			e.GetServer().Remove(record.GetCacheKey())
		}

		return rsp, header.RspReportDownloadResult
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}

// Validate
func (e *ReportDownloadResult) Validate(req *protos.ReqReportDownloadResult) (bool, string) {

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
	//if e.GetServer().CT.Fetch(user) != nil {
	//	return false, "not authorized to process"
	//}
	//
	//pukInByte, err := hex.DecodeString(user.Puk)
	//if err != nil {
	//	return false, err.Error()
	//}
	//
	//puk, err := crypto.UnmarshalPubkey(pukInByte)
	//if err != nil {
	//	return false, err.Error()
	//}
	//
	//data := req.MyWalletAddress + req.FileHash
	//if !utils.ECCVerify([]byte(data), req.Sign, puk) {
	//	return false, "signature verification failed"
	//}

	return true, ""
}
