package event

// Author j cc
import (
	"context"
	"fmt"
	"net/http"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/encryption"
	"github.com/stratosnet/sds/utils/encryption/hdkey"
)

const (
	LOSE_SLICE_MSG = "cannot find the file slice"
)

// ReqDownloadSlice download slice PP-storagePP
func ReqDownloadSlice(ctx context.Context, conn core.WriteCloser) {
	utils.Log("ReqDownloadSlice", conn)
	var target protos.ReqDownloadSlice
	reqId := requests.GetReqIdFromMessage(ctx)
	if requests.UnmarshalData(ctx, &target) {
		rsp := requests.RspDownloadSliceData(&target)

		if task.DownloadSliceTaskMap.HashKey(target.TaskId + target.SliceInfo.SliceHash) {
			rsp.Data = nil
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "duplicate request for the same slice in the same download task"
			peers.SendResponseMessageWithReqId(conn, rsp, header.RspDownloadSlice, reqId)
			return
		}

		if target.Sign == nil || !verifyDownloadSliceSign(&target, rsp) {
			rsp.Data = nil
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "signature validation failed"
			peers.SendResponseMessageWithReqId(conn, rsp, header.RspDownloadSlice, reqId)
			return
		}

		if rsp.SliceSize == 0 {
			utils.DebugLog("cannot find slice, sliceHash: ", target.SliceInfo.SliceHash)
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = LOSE_SLICE_MSG
			peers.SendResponseMessageWithReqId(conn, rsp, header.RspDownloadSlice, reqId)
			return
		}

		SendReportDownloadResult(rsp, true)
		splitSendDownloadSliceData(rsp, conn, reqId)

		task.DownloadSliceTaskMap.Store(target.TaskId+target.SliceInfo.SliceHash, true)
	}
}

func splitSendDownloadSliceData(rsp *protos.RspDownloadSlice, conn core.WriteCloser, reqId int64) {
	dataLen := uint64(len(rsp.Data))
	utils.DebugLog("dataLen=========", dataLen)
	dataStart := uint64(0)
	dataEnd := uint64(setting.MAXDATA)
	offsetStart := rsp.SliceInfo.SliceOffset.SliceOffsetStart
	offsetEnd := rsp.SliceInfo.SliceOffset.SliceOffsetStart + dataEnd
	for {
		utils.DebugLog("_____________________________")
		utils.DebugLog(dataStart, dataEnd, offsetStart, offsetEnd)
		if dataEnd < dataLen {
			peers.SendResponseMessageWithReqId(conn, requests.RspDownloadSliceDataSplit(rsp, dataStart, dataEnd, offsetStart, offsetEnd,
				rsp.SliceInfo.SliceOffset.SliceOffsetStart, rsp.SliceInfo.SliceOffset.SliceOffsetEnd, false), header.RspDownloadSlice, reqId)
			dataStart += setting.MAXDATA
			dataEnd += setting.MAXDATA
			offsetStart += setting.MAXDATA
			offsetEnd += setting.MAXDATA
		} else {
			peers.SendResponseMessageWithReqId(conn, requests.RspDownloadSliceDataSplit(rsp, dataStart, 0, offsetStart, 0,
			rsp.SliceInfo.SliceOffset.SliceOffsetStart, rsp.SliceInfo.SliceOffset.SliceOffsetEnd, true), header.RspDownloadSlice, reqId)
			return
		}
	}
}

// RspDownloadSlice storagePP-PP
func RspDownloadSlice(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get RspDownloadSlice")
	var target protos.RspDownloadSlice
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	sid, ok := task.SliceSessionMap.Load(target.ReqId)
	if !ok {
		utils.DebugLog("Can't find who created slice request", target.ReqId)
	}

	dTask, ok := task.GetDownloadTask(target.FileHash, target.WalletAddress, sid.(string))
	if !ok {
		utils.DebugLog("current task is stopped！！！！！！！！！！！！！！！！！！！！！！！！！！")
		return
	}

	if target.SliceSize <= 0 || (target.Result.State == protos.ResultState_RES_FAIL && target.Result.Msg == LOSE_SLICE_MSG) {
		utils.DebugLog("slice was not found, will send msg to sp for retry, sliceHash: ", target.SliceInfo.SliceHash)
		setDownloadSliceFail(target.SliceInfo.SliceHash, dTask)
		return
	}

	if target.Result.State == protos.ResultState_RES_FAIL {
		utils.ErrorLog(target.Result.Msg)
		return
	}

	if f, ok := task.DownloadFileMap.Load(target.FileHash+sid.(string)); ok {
		fInfo := f.(*protos.RspFileStorageInfo)
		utils.DebugLog("get a slice -------")
		utils.DebugLog("SliceHash", target.SliceInfo.SliceHash)
		utils.DebugLog("SliceOffset", target.SliceInfo.SliceOffset)
		utils.DebugLog("length", len(target.Data))
		utils.DebugLog("sliceSize", target.SliceSize)
		if fInfo.EncryptionTag != "" {
			receiveSliceAndProgressEncrypted(&target, fInfo, dTask)
		} else {
			receiveSliceAndProgress(&target, fInfo, dTask)
		}
		if !fInfo.IsVideoStream {
			task.DownloadProgress(target.FileHash, sid.(string), uint64(len(target.Data)))
		}
	} else {
		utils.DebugLog("DownloadFileMap doesn't have entry with file hash", target.FileHash);
	}
}

func receiveSliceAndProgress(target *protos.RspDownloadSlice, fInfo *protos.RspFileStorageInfo, dTask *task.DownloadTask) {
	if task.SaveDownloadFile(target, fInfo) {
		dataLen := uint64(len(target.Data))
		if s, ok := task.DownloadSliceProgress.Load(target.SliceInfo.SliceHash + fInfo.ReqId); ok {
			alreadySize := s.(uint64)
			alreadySize += dataLen
			if alreadySize == target.SliceSize {
				utils.DebugLog("slice download finished", target.SliceInfo.SliceHash)
				task.DownloadSliceProgress.Delete(target.SliceInfo.SliceHash + fInfo.ReqId)
				receivedSlice(target, fInfo, dTask)
			} else {
				task.DownloadSliceProgress.Store(target.SliceInfo.SliceHash + fInfo.ReqId, alreadySize)
			}
		} else {
			// if data is sent at once
			if target.SliceSize == dataLen {
				receivedSlice(target, fInfo, dTask)
			} else {
				task.DownloadSliceProgress.Store(target.SliceInfo.SliceHash + fInfo.ReqId, dataLen)
			}
		}
	}else {
		utils.DebugLog("Download failed: not able to write to the target file.")
		file.CloseDownloadSession(fInfo.FileHash + fInfo.ReqId)
		task.CleanDownloadFileAndConnMap(fInfo.FileHash, fInfo.ReqId)
	}
}

func receiveSliceAndProgressEncrypted(target *protos.RspDownloadSlice, fInfo *protos.RspFileStorageInfo, dTask *task.DownloadTask) {
	dataToDecrypt := target.Data
	dataToDecryptSize := uint64(len(dataToDecrypt))
	encryptedOffset := target.SliceInfo.EncryptedSliceOffset

	if existingSlice, ok := task.DownloadEncryptedSlices.Load(target.SliceInfo.SliceHash + fInfo.ReqId); ok {
		existingSliceData := existingSlice.([]byte)
		copy(existingSliceData[encryptedOffset.SliceOffsetStart:encryptedOffset.SliceOffsetEnd], dataToDecrypt)
		dataToDecrypt = existingSliceData

		if s, ok := task.DownloadSliceProgress.Load(target.SliceInfo.SliceHash + fInfo.ReqId); ok {
			existingSize := s.(uint64)
			dataToDecryptSize += existingSize
		}
	}

	if dataToDecryptSize >= target.SliceSize {
		// Decrypt slice data and save it to file
		decryptedData, err := decryptSliceData(dataToDecrypt)
		if err != nil {
			utils.ErrorLog("Couldn't decrypt slice", err)
			return
		}
		target.Data = decryptedData

		if task.SaveDownloadFile(target, fInfo) {
			utils.DebugLog("slice download finished", target.SliceInfo.SliceHash)
			task.DownloadSliceProgress.Delete(target.SliceInfo.SliceHash + fInfo.ReqId)
			task.DownloadEncryptedSlices.Delete(target.SliceInfo.SliceHash + fInfo.ReqId)
			receivedSlice(target, fInfo, dTask)
		}
	} else {
		// Store partial slice data to memory
		dataToStore := dataToDecrypt
		if uint64(len(dataToStore)) < target.SliceSize {
			dataToStore = make([]byte, target.SliceSize)
			copy(dataToStore[encryptedOffset.SliceOffsetStart:encryptedOffset.SliceOffsetEnd], dataToDecrypt)
		}
		task.DownloadEncryptedSlices.Store(target.SliceInfo.SliceHash + fInfo.ReqId, dataToStore)
		task.DownloadSliceProgress.Store(target.SliceInfo.SliceHash + fInfo.ReqId, dataToDecryptSize)
	}
}

func receivedSlice(target *protos.RspDownloadSlice, fInfo *protos.RspFileStorageInfo, dTask *task.DownloadTask) {
	file.SaveDownloadProgress(target.SliceInfo.SliceHash, fInfo.FileName, target.FileHash, target.SavePath, fInfo.ReqId)
	task.CleanDownloadTask(target.FileHash, target.SliceInfo.SliceHash, target.WalletAddress, fInfo.ReqId)
	target.Result = &protos.Result{
		State: protos.ResultState_RES_SUCCESS,
	}
	if fInfo.IsVideoStream && !target.IsVideoCaching {
		putData(target.ReqId, HTTPDownloadSlice, target)
	} else if fInfo.IsVideoStream && target.IsVideoCaching {
		videoCacheKeep(fInfo.FileHash, target.TaskId)
	}
	setDownloadSliceSuccess(target.SliceInfo.SliceHash, dTask)
	SendReportDownloadResult(target, false)
}

func videoCacheKeep(fileHash, taskID string) {
	utils.DebugLogf("download keep fileHash = %v  taskID = %v", fileHash, taskID)
	if ing, ok := task.VideoCacheTaskMap.Load(fileHash); ok {
		ING := ing.(*task.VideoCacheTask)
		ING.DownloadCh <- true
	}
}

// ReportDownloadResult  PP-SP OR StoragePP-SP
func SendReportDownloadResult(target *protos.RspDownloadSlice, isPP bool) {
	utils.DebugLog("ReportDownloadResult report result target.fileHash = ", target.FileHash)
	peers.SendMessageDirectToSPOrViaPP(requests.ReqReportDownloadResultData(target, isPP), header.ReqReportDownloadResult)
}

// ReportDownloadResult  P-SP OR PP-SP
func SendReportStreamingResult(target *protos.RspDownloadSlice, isPP bool) {
	peers.SendMessageToSPServer(requests.ReqReportDownloadResultData(target, isPP), header.ReqReportDownloadResult)
}

// DownloadFileSlice
func DownloadFileSlice(target *protos.RspFileStorageInfo) {
	fileSize := uint64(0)

	dTask, _ := task.GetDownloadTask(target.FileHash, target.WalletAddress, target.ReqId)
	for _, sliceInfo := range target.SliceInfo {
		fileSize += sliceInfo.SliceStorageInfo.SliceSize
	}
	utils.DebugLog(fmt.Sprintf("file size: %v  raw file size: %v\n", fileSize, target.FileSize))

	sp := &task.DownloadSP{
		RawSize:        int64(target.FileSize),
		TotalSize:      int64(fileSize),
		DownloadedSize: 0,
	}
	if !file.CheckFileExisting(target.FileHash, target.FileName, target.SavePath, target.EncryptionTag, target.ReqId) {
		task.DownloadSpeedOfProgress.Store(target.FileHash + target.ReqId, sp)
		for _, rsp := range target.SliceInfo {
			utils.DebugLog("taskid ======= ", rsp.TaskId)
			if file.CheckSliceExisting(target.FileHash, target.FileName, rsp.SliceStorageInfo.SliceHash, target.SavePath, target.ReqId) {
				utils.Log("slice exist already,", rsp.SliceStorageInfo.SliceHash)
				task.DownloadProgress(target.FileHash, target.ReqId, rsp.SliceStorageInfo.SliceSize)
				task.CleanDownloadTask(target.FileHash, rsp.SliceStorageInfo.SliceHash, target.WalletAddress, target.ReqId)
				setDownloadSliceSuccess(rsp.SliceStorageInfo.SliceHash, dTask)
			} else {
				utils.DebugLog("request download data")
				req := requests.ReqDownloadSliceData(target, rsp)
				task.SliceSessionMap.Store(req.ReqId, target.ReqId)
				SendReqDownloadSlice(target.FileHash, rsp, req, target.ReqId)
			}
		}
  	} else {
		utils.ErrorLog("file exists already!")
		task.DeleteDownloadTask(target.FileHash, target.WalletAddress, target.ReqId)
	}
}

func SendReqDownloadSlice(fileHash string, sliceInfo *protos.DownloadSliceInfo, req *protos.ReqDownloadSlice, fileReqId string) {
	utils.DebugLog("req = ", req)
	
	networkAddress := sliceInfo.StoragePpInfo.NetworkAddress
	key := fileHash + sliceInfo.StoragePpInfo.P2PAddress + fileReqId

	if c, ok := client.DownloadConnMap.Load(key); ok {
		conn := c.(*cf.ClientConn)
		err := peers.SendMessage(conn, req, header.ReqDownloadSlice)
		if err == nil {
			utils.DebugLog("Send download slice request to ", networkAddress)
			return
		}
	}

	if conn, ok := client.ConnMap[networkAddress]; ok {
		err := peers.SendMessage(conn, req, header.ReqDownloadSlice)
		if err == nil {
			utils.DebugLog("Send download slice request to ", networkAddress)
			client.DownloadConnMap.Store(key, conn)
			return
		}
	}

	conn := client.NewClient(networkAddress, false)
	if conn == nil {
		utils.ErrorLog("Fail to create connection with " + networkAddress)
		if dTask, ok := task.GetDownloadTask(fileHash, req.WalletAddress, fileReqId); ok {
			setDownloadSliceFail(sliceInfo.SliceStorageInfo.SliceHash, dTask)
		}
		return
	}

	err := peers.SendMessage(conn, req, header.ReqDownloadSlice)
	if err == nil {
		utils.DebugLog("Send download slice request to ", networkAddress)
		client.DownloadConnMap.Store(key, conn)
	} else {
		utils.ErrorLog("Fail to send download slice request to" + networkAddress)
	}
}

// RspReportDownloadResult  SP-P OR SP-PP
func RspReportDownloadResult(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get RspReportDownloadResult")
	var target protos.RspReportDownloadResult
	if requests.UnmarshalData(ctx, &target) {
		utils.DebugLog("result", target.Result.State, target.Result.Msg)
	}
}

// RspDownloadSliceWrong
func RspDownloadSliceWrong(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("RspDownloadSlice")
	var target protos.RspDownloadSliceWrong
	if requests.UnmarshalData(ctx, &target) {
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			utils.DebugLog("RspDownloadSliceWrongRspDownloadSliceWrongRspDownloadSliceWrong", target.NewSliceInfo.SliceStorageInfo.SliceHash)
			if dlTask, ok := task.DownloadTaskMap.Load(target.FileHash + target.WalletAddress + task.LOCAL_REQID); ok {
				downloadTask := dlTask.(*task.DownloadTask)
				if sInfo, ok := downloadTask.SliceInfo[target.NewSliceInfo.SliceStorageInfo.SliceHash]; ok {
					sInfo.StoragePpInfo.P2PAddress = target.NewSliceInfo.StoragePpInfo.P2PAddress
					sInfo.StoragePpInfo.WalletAddress = target.NewSliceInfo.StoragePpInfo.WalletAddress
					sInfo.StoragePpInfo.NetworkAddress = target.NewSliceInfo.StoragePpInfo.NetworkAddress
					peers.TransferSendMessageToPPServ(target.NewSliceInfo.StoragePpInfo.NetworkAddress, requests.RspDownloadSliceWrong(&target))
				}
			}
		}
	}
}

func downloadWrong(taskID, sliceHash, p2pAddress, walletAddress string, wrongType protos.DownloadWrongType) {
	utils.DebugLog("downloadWrong, sliceHash: ", sliceHash)
	peers.SendMessageToSPServer(requests.ReqDownloadSliceWrong(taskID, sliceHash, p2pAddress, walletAddress, wrongType), header.ReqDownloadSliceWrong)
}

// DownloadSlicePause
func DownloadSlicePause(fileHash, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		// storeResponseWriter(reqID, w)
		task.DownloadTaskMap.Delete(fileHash + setting.WalletAddress + task.LOCAL_REQID)
		task.CleanDownloadFileAndConnMap(fileHash, reqID)
	} else {
		notLogin(w)
	}
}

// DownloadSliceCancel
func DownloadSliceCancel(fileHash, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		storeResponseWriter(reqID, w)
		task.DownloadTaskMap.Delete(fileHash + setting.WalletAddress + task.LOCAL_REQID)
		task.CleanDownloadFileAndConnMap(fileHash, reqID)
		task.CancelDownloadTask(fileHash)
	} else {
		notLogin(w)
	}
}

func decryptSliceData(dataToDecrypt []byte) ([]byte, error) {
	encryptedSlice := protos.EncryptedSlice{}
	err := proto.Unmarshal(dataToDecrypt, &encryptedSlice)
	if err != nil {
		utils.ErrorLog("Couldn't unmarshal protobuf to encrypted slice", err)
		return nil, err
	}

	key, err := hdkey.MasterKeyForSliceEncryption(setting.WalletPrivateKey, encryptedSlice.HdkeyNonce)
	if err != nil {
		utils.ErrorLog("Couldn't generate slice encryption master key", err)
		return nil, err
	}

	return encryption.DecryptAES(key.PrivateKey(), encryptedSlice.Data, encryptedSlice.AesNonce)
}

func verifyDownloadSliceSign(target *protos.ReqDownloadSlice, rsp *protos.RspDownloadSlice) bool {
	if !requests.VerifySpSignature(target.SpP2PAddress, []byte(target.P2PAddress+target.FileHash+header.ReqDownloadSlice), target.Sign) {
		return false
	}

	return target.SliceInfo.SliceHash == utils.CalcSliceHash(rsp.Data, target.FileHash, target.SliceNumber)
}

func setDownloadSliceSuccess(sliceHash string, dTask *task.DownloadTask) {
	dTask.SetSliceSuccess(sliceHash)
	CheckAndSendRetryMessage(dTask)
}

func setDownloadSliceFail(sliceHash string, dTask *task.DownloadTask) {
	dTask.AddFailedSlice(sliceHash)
	CheckAndSendRetryMessage(dTask)
}
