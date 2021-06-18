package events

import (
	"context"
	"encoding/hex"
	"path/filepath"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/common"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/data"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
)

// reportUploadSliceResult is a concrete implementation of event
type reportUploadSliceResult struct {
	event
}

const reportUploadSliceResultEvent = "report_upload_slice_result"

// GetReportUploadSliceResultHandler creates event and return handler func for it
func GetReportUploadSliceResultHandler(s *net.Server) EventHandleFunc {
	e := reportUploadSliceResult{newEvent(reportUploadSliceResultEvent, s, reportUploadSliceResultCallbackFunc)}
	return e.Handle
}

// reportUploadSliceResultCallbackFunc is the main process of report upload slice result
func reportUploadSliceResultCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReportUploadSliceResult)

	rsp := &protos.RspReportUploadSliceResult{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		SliceNumAddr: &protos.SliceNumAddr{
			PpInfo: &protos.PPBaseInfo{
				WalletAddress:  body.SliceNumAddr.PpInfo.WalletAddress,
				NetworkAddress: body.SliceNumAddr.PpInfo.NetworkAddress,
			},
			SliceNumber: body.SliceNumAddr.SliceNumber,
		},
	}

	if ok, msg := validateReportUploadSliceResultRequest(body, s); !ok {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = msg
		return rsp, header.RspReportUploadSliceResult
	}

	fileSlice := &table.FileSlice{
		FileSliceStorage: table.FileSliceStorage{
			WalletAddress: body.SliceNumAddr.PpInfo.WalletAddress,
		},
		SliceHash: body.SliceHash,
		TaskId:    body.TaskId,
	}

	traffic := &table.Traffic{TaskId: body.TaskId}

	//todo change to read from redis
	s.Lock()
	if s.Load(fileSlice) == nil {

		if fileSlice.Status == table.FILE_SLICE_STATUS_SUCCESS {
			//skip because success
			s.Unlock()
			return rsp, header.RspReportUploadSliceResult
		}

		fileSlice.Status = table.FILE_SLICE_STATUS_SUCCESS
		fileSlice.Time = time.Now().Unix()

		// validate report result
		if fileSlice.SliceSize != body.SliceSize ||
			fileSlice.SliceNumber != body.SliceNumAddr.SliceNumber ||
			fileSlice.NetworkAddress != body.SliceNumAddr.PpInfo.NetworkAddress ||
			fileSlice.WalletAddress != body.SliceNumAddr.PpInfo.WalletAddress ||
			fileSlice.FileHash != body.FileHash {

			rsp.Result.Msg = "report result validate failed"
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.SliceNumAddr = nil

			s.Unlock()
			return rsp, header.RspReportUploadSliceResult
		}

		if body.IsPP {
			// todo if is pp do something
		}

	} else {

		fileSlice.FileHash = body.FileHash
		fileSlice.SliceHash = body.SliceHash
		fileSlice.SliceSize = body.SliceSize
		fileSlice.SliceNumber = body.SliceNumAddr.SliceNumber
		fileSlice.SliceOffsetStart = body.SliceNumAddr.SliceOffset.SliceOffsetStart
		fileSlice.SliceOffsetEnd = body.SliceNumAddr.SliceOffset.SliceOffsetEnd
		fileSlice.WalletAddress = body.SliceNumAddr.PpInfo.WalletAddress
		fileSlice.NetworkAddress = body.SliceNumAddr.PpInfo.NetworkAddress
		fileSlice.Status = table.FILE_SLICE_STATUS_CHECK
		fileSlice.Time = time.Now().Unix()
	}

	s.Load(traffic)
	if body.IsPP {
		traffic.ProviderWalletAddress = body.WalletAddress
	} else {
		traffic.ConsumerWalletAddress = body.WalletAddress
	}

	// store file slice info todo change to read from redis
	if err := s.Store(fileSlice, 3600*time.Second); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, reportUploadSliceResultEvent, "store file slice 1", err)
	}

	if err := s.Store(traffic, 3600*time.Second); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, reportUploadSliceResultEvent, "store traffic record to cache", err)
	}

	s.Unlock()

	// query file upload info
	uploadFile := &data.UploadFile{
		Key: body.TaskId,
	}
	if s.Load(uploadFile) != nil {
		return rsp, header.RspReportUploadSliceResult
	}
	if fileSlice.Status != table.FILE_SLICE_STATUS_SUCCESS {
		return rsp, header.RspReportUploadSliceResult
	}

	if err := s.CT.Save(fileSlice); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, reportUploadSliceResultEvent, "save file slice", err)
	}

	traffic.Volume = fileSlice.SliceSize
	traffic.TaskType = table.TRAFFIC_TASK_TYPE_UPLOAD
	traffic.DeliveryTime = time.Now().Unix()

	if ok, err := s.CT.StoreTable(traffic); !ok {
		utils.ErrorLogf(eventHandleErrorTemplate, reportUploadSliceResultEvent, "store traffic record table to db", err)
	}

	uploadFile.SetSliceFinish(fileSlice.SliceNumber)
	if err := s.Store(uploadFile, 3600*time.Second); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, reportUploadSliceResultEvent, "store file slice 2", err)
	}

	// check if all slice upload finished
	if uploadFile.IsUploadFinished() {

		file := &table.File{
			UserHasFile: table.UserHasFile{
				WalletAddress: uploadFile.WalletAddress,
			},
			Hash: uploadFile.FileHash,
		}

		if s.CT.Fetch(file) != nil || file.State == table.STATE_DELETE {
			file.Name = uploadFile.FileName
			if len(uploadFile.FileName) > 128 {
				suffix := filepath.Ext(uploadFile.FileName)
				file.Name = uploadFile.FileName[0:(128-len(suffix))] + suffix
			}
			file.Hash = uploadFile.FileHash
			file.Size = uploadFile.FileSize
			file.SliceNum = uploadFile.SliceNum
			file.WalletAddress = uploadFile.WalletAddress
			file.State = table.STATE_OK
			file.Time = time.Now().Unix()

			if uploadFile.IsCover {
				file.IsCover = table.IS_COVER
			}

			if s.CT.Save(file) == nil {
				if uploadFile.FilePath != "" {
					dirMapFile := new(table.UserDirectoryMapFile)
					dirMapFile.WalletAddress = uploadFile.WalletAddress
					dirMapFile.Path = uploadFile.FilePath
					dirMapFile.FileHash = file.Hash
					dirMapFile.DirHash = dirMapFile.GenericHash()
					dirMapFile.Owner = uploadFile.WalletAddress
					if _, err := s.CT.InsertTable(dirMapFile); err != nil {
						utils.ErrorLogf(eventHandleErrorTemplate, reportUploadSliceResultEvent, "insert dir map", err)
					}
				}
			}

			if err := s.Remove(uploadFile.GetCacheKey()); err != nil {
				utils.ErrorLogf(eventHandleErrorTemplate, reportUploadSliceResultEvent, "remove upload file", err)
			}
		}
	}

	// if upload finish, started backup
	backupSliceMsg := &common.MsgBackupSlice{
		SliceHash:         fileSlice.SliceHash,
		FromWalletAddress: fileSlice.WalletAddress,
	}
	s.HandleMsg(backupSliceMsg)

	return rsp, header.RspReportUploadSliceResult
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *reportUploadSliceResult) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReportUploadSliceResult{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}

// validateReportUploadSliceResultRequest checks request
func validateReportUploadSliceResultRequest(req *protos.ReportUploadSliceResult, s *net.Server) (bool, string) {

	if req.FileHash == "" || req.SliceHash == "" || req.SliceNumAddr.SliceNumber <= 0 ||
		req.SliceNumAddr.PpInfo.WalletAddress == "" || req.SliceNumAddr.PpInfo.NetworkAddress == "" {
		return false, "slice info invalid"
	}

	if req.TaskId == "" {
		return false, "task ID can't be empty"
	}

	if req.WalletAddress == "" {
		return false, "wallet address can't be empty"
	}

	if len(req.Sign) <= 0 {
		return false, "signature can't be empty"
	}

	user := &table.User{WalletAddress: req.WalletAddress}
	if s.CT.Fetch(user) != nil {
		return false, "not authorized to process"
	}

	puk, err := hex.DecodeString(user.Puk)
	if err != nil {
		return false, err.Error()
	}

	d := req.WalletAddress + req.FileHash
	if !utils.ECCVerifyBytes([]byte(d), req.Sign, puk) {
		return false, "signature verification failed"
	}

	return true, ""
}
