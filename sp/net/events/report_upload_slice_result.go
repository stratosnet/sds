package events

import (
	"context"
	"encoding/hex"
	"path/filepath"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/common"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/data"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto"
	"time"
)

// ReportUploadSliceResult
type ReportUploadSliceResult struct {
	Server *net.Server
}

// GetServer
func (e *ReportUploadSliceResult) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *ReportUploadSliceResult) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *ReportUploadSliceResult) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReportUploadSliceResult)

	callback := func(message interface{}) (interface{}, string) {

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


		if ok, msg := e.Validate(body); !ok {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = msg
			return rsp, header.RspReportUploadSliceResult
		}


		fileSlice := new(table.FileSlice)

		fileSlice.SliceHash = body.SliceHash
		fileSlice.WalletAddress = body.SliceNumAddr.PpInfo.WalletAddress
		fileSlice.TaskId = body.TaskId

		//todo change to read from redis
		e.GetServer().Lock()
		if e.GetServer().Load(fileSlice) == nil {

			if fileSlice.Status == table.FILE_SLICE_STATUS_SUCCESS {
				//skip because success
				e.GetServer().Unlock()
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

				e.GetServer().Unlock()
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

		// store file slice info todo change to read from redis
		e.GetServer().Store(fileSlice, 3600*time.Second)

		e.GetServer().Unlock()

		// query file upload info
		uploadFile := &data.UploadFile{
			Key: body.TaskId,
		}
		if e.GetServer().Load(uploadFile) == nil {
			if fileSlice.Status == table.FILE_SLICE_STATUS_SUCCESS {

				e.GetServer().CT.Save(fileSlice)


				uploadFile.SetSliceFinish(fileSlice.SliceNumber)
				e.GetServer().Store(uploadFile, 3600*time.Second)

				// check if all slice upload finished
				if uploadFile.IsUploadFinished() {


					file := new(table.File)
					file.Hash = uploadFile.FileHash
					file.WalletAddress = uploadFile.WalletAddress

					if e.GetServer().CT.Fetch(file) != nil || file.State == table.STATE_DELETE {
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

						if e.GetServer().CT.Save(file) == nil {
							if uploadFile.FilePath != "" {
								dirMapFile := new(table.UserDirectoryMapFile)
								dirMapFile.WalletAddress = uploadFile.WalletAddress
								dirMapFile.Path = uploadFile.FilePath
								dirMapFile.FileHash = file.Hash
								dirMapFile.DirHash = dirMapFile.GenericHash()
								dirMapFile.Owner = uploadFile.WalletAddress
								e.GetServer().CT.InsertTable(dirMapFile)
							}
						}


						e.GetServer().Remove(uploadFile.GetCacheKey())
					}
				}

				// if upload finish, started backup
				backupSliceMsg := &common.MsgBackupSlice{
					SliceHash:         fileSlice.SliceHash,
					FromWalletAddress: fileSlice.WalletAddress,
				}
				e.GetServer().HandleMsg(backupSliceMsg)
			}
		}

		return rsp, header.RspReportUploadSliceResult
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}

// Validate
func (e *ReportUploadSliceResult) Validate(req *protos.ReportUploadSliceResult) (bool, string) {


	if req.FileHash == "" ||
		req.SliceHash == "" ||
		req.SliceNumAddr.SliceNumber <= 0 ||
		req.SliceNumAddr.PpInfo.WalletAddress == "" ||
		req.SliceNumAddr.PpInfo.NetworkAddress == "" {

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
	if e.GetServer().CT.Fetch(user) != nil {
		return false, "not authorized to process"
	}

	pukInByte, err := hex.DecodeString(user.Puk)
	if err != nil {
		return false, err.Error()
	}

	puk, err := crypto.UnmarshalPubkey(pukInByte)
	if err != nil {
		return false, err.Error()
	}

	data := req.WalletAddress + req.FileHash
	if !utils.ECCVerify([]byte(data), req.Sign, puk) {
		return false, "signature verification failed"
	}

	return true, ""
}
