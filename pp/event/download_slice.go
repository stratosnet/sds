package event

// Author j cc
import (
	"context"
	ed25519crypto "crypto/ed25519"
	"fmt"
	"net/http"

	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/relay"
	tmed25519 "github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/bech32"

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

// ReqDownloadSlice download slice PP-storagePP
func ReqDownloadSlice(ctx context.Context, conn core.WriteCloser) {
	utils.Log("ReqDownloadSlice", conn)
	var target protos.ReqDownloadSlice
	if requests.UnmarshalData(ctx, &target) {
		rsp := requests.RspDownloadSliceData(&target)
		if target.Sign == nil || !verifySignature(&target, rsp) {
			rsp.Data = nil
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "signature validation failed"
			peers.SendMessage(conn, rsp, header.RspDownloadSlice)
		}
		if rsp.SliceSize > 0 {
			SendReportDownloadResult(rsp, true)
			splitSendDownloadSliceData(rsp, conn)
		} else {
			downloadWrong(target.TaskId, target.SliceInfo.SliceHash, target.P2PAddress, target.WalletAddress, protos.DownloadWrongType_LOSESLICE)
		}
	}
}

func splitSendDownloadSliceData(rsp *protos.RspDownloadSlice, conn core.WriteCloser) {
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
			peers.SendMessage(conn, requests.RspDownloadSliceDataSplit(rsp, dataStart, dataEnd, offsetStart, offsetEnd,
				rsp.SliceInfo.SliceOffset.SliceOffsetStart, rsp.SliceInfo.SliceOffset.SliceOffsetEnd, false), header.RspDownloadSlice)
			dataStart += setting.MAXDATA
			dataEnd += setting.MAXDATA
			offsetStart += setting.MAXDATA
			offsetEnd += setting.MAXDATA
		} else {
			peers.SendMessage(conn, requests.RspDownloadSliceDataSplit(rsp, dataStart, 0, offsetStart, 0,
				rsp.SliceInfo.SliceOffset.SliceOffsetStart, rsp.SliceInfo.SliceOffset.SliceOffsetEnd, true), header.RspDownloadSlice)
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

	if target.Result.State == protos.ResultState_RES_FAIL {
		utils.ErrorLog(target.Result.Msg)
		return
	}

	if _, ok := task.DownloadTaskMap.Load(target.FileHash + target.WalletAddress); !ok {
		utils.DebugLog("current task is stopped！！！！！！！！！！！！！！！！！！！！！！！！！！")
		return
	}

	if target.SliceSize <= 0 {
		downloadWrong(target.TaskId, target.SliceInfo.SliceHash, target.P2PAddress, target.WalletAddress, protos.DownloadWrongType_LOSESLICE)
		return
	}

	if f, ok := task.DownloadFileMap.Load(target.FileHash); ok {
		fInfo := f.(*protos.RspFileStorageInfo)
		utils.DebugLog("get a slice -------")
		utils.DebugLog("SliceHash", target.SliceInfo.SliceHash)
		utils.DebugLog("SliceOffset", target.SliceInfo.SliceOffset)
		utils.DebugLog("length", len(target.Data))
		utils.DebugLog("sliceSize", target.SliceSize)
		if fInfo.EncryptionTag != "" {
			receiveSliceAndProgressEncrypted(&target, fInfo)
		} else {
			receiveSliceAndProgress(&target, fInfo)
		}
		if !fInfo.IsVideoStream {
			task.DownloadProgress(target.FileHash, uint64(len(target.Data)))
		}
	}
}

func receiveSliceAndProgress(target *protos.RspDownloadSlice, fInfo *protos.RspFileStorageInfo) {
	if task.SaveDownloadFile(target, fInfo) {
		dataLen := uint64(len(target.Data))
		if s, ok := task.DownloadSliceProgress.Load(target.SliceInfo.SliceHash); ok {
			alreadySize := s.(uint64)
			alreadySize += dataLen
			if alreadySize == target.SliceSize {
				utils.DebugLog("slice download finished", target.SliceInfo.SliceHash)
				task.DownloadSliceProgress.Delete(target.SliceInfo.SliceHash)
				receivedSlice(target, fInfo)
			} else {
				task.DownloadSliceProgress.Store(target.SliceInfo.SliceHash, alreadySize)
			}
		} else {
			// if data is sent at once
			if target.SliceSize == dataLen {
				receivedSlice(target, fInfo)
			} else {
				task.DownloadSliceProgress.Store(target.SliceInfo.SliceHash, dataLen)
			}
		}
	}
}

func receiveSliceAndProgressEncrypted(target *protos.RspDownloadSlice, fInfo *protos.RspFileStorageInfo) {
	dataToDecrypt := target.Data
	dataToDecryptSize := uint64(len(dataToDecrypt))
	encryptedOffset := target.SliceInfo.EncryptedSliceOffset

	if existingSlice, ok := task.DownloadEncryptedSlices.Load(target.SliceInfo.SliceHash); ok {
		existingSliceData := existingSlice.([]byte)
		copy(existingSliceData[encryptedOffset.SliceOffsetStart:encryptedOffset.SliceOffsetEnd], dataToDecrypt)
		dataToDecrypt = existingSliceData

		if s, ok := task.DownloadSliceProgress.Load(target.SliceInfo.SliceHash); ok {
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
			task.DownloadSliceProgress.Delete(target.SliceInfo.SliceHash)
			task.DownloadEncryptedSlices.Delete(target.SliceInfo.SliceHash)
			receivedSlice(target, fInfo)
		}
	} else {
		// Store partial slice data to memory
		dataToStore := dataToDecrypt
		if uint64(len(dataToStore)) < target.SliceSize {
			dataToStore = make([]byte, target.SliceSize)
			copy(dataToStore[encryptedOffset.SliceOffsetStart:encryptedOffset.SliceOffsetEnd], dataToDecrypt)
		}
		task.DownloadEncryptedSlices.Store(target.SliceInfo.SliceHash, dataToStore)
		task.DownloadSliceProgress.Store(target.SliceInfo.SliceHash, dataToDecryptSize)
	}
}

func receivedSlice(target *protos.RspDownloadSlice, fInfo *protos.RspFileStorageInfo) {
	file.SaveDownloadProgress(target.SliceInfo.SliceHash, fInfo.FileName, target.FileHash, target.SavePath)
	task.CleanDownloadTask(target.FileHash, target.SliceInfo.SliceHash, target.WalletAddress)
	target.Result = &protos.Result{
		State: protos.ResultState_RES_SUCCESS,
	}
	if fInfo.IsVideoStream && !target.IsVideoCaching {
		putData(target.ReqId, HTTPDownloadSlice, target)
	} else if fInfo.IsVideoStream && target.IsVideoCaching {
		videoCacheKeep(fInfo.FileHash, target.TaskId)
	}
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
	for _, sliceInfo := range target.SliceInfo {
		fileSize += sliceInfo.SliceStorageInfo.SliceSize
	}
	utils.DebugLog(fmt.Sprintf("file size: %v  raw file size: %v\n", fileSize, target.FileSize))

	sp := &task.DownloadSP{
		RawSize:        int64(target.FileSize),
		TotalSize:      int64(fileSize),
		DownloadedSize: 0,
	}
	task.DownloadSpeedOfProgress.Store(target.FileHash, sp)
	if !file.CheckFileExisting(target.FileHash, target.FileName, target.SavePath, target.EncryptionTag) {
		for _, rsp := range target.SliceInfo {
			utils.DebugLog("taskid ======= ", rsp.TaskId)
			if file.CheckSliceExisting(target.FileHash, target.FileName, rsp.SliceStorageInfo.SliceHash, target.SavePath) {
				utils.Log("slice exist already,", rsp.SliceStorageInfo.SliceHash)
				task.DownloadProgress(target.FileHash, rsp.SliceStorageInfo.SliceSize)
			} else {
				utils.DebugLog("request download data")
				req := requests.ReqDownloadSliceData(target, rsp)
				SendReqDownloadSlice(target.FileHash, rsp, req)
			}
		}
	} else {
		utils.ErrorLog("file exists already!")
	}
}

func SendReqDownloadSlice(fileHash string, sliceInfo *protos.DownloadSliceInfo, req *protos.ReqDownloadSlice) {
	utils.DebugLog("req = ", req)

	networkAddress := sliceInfo.StoragePpInfo.NetworkAddress
	key := fileHash + sliceInfo.StoragePpInfo.P2PAddress

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
			if dlTask, ok := task.DownloadTaskMap.Load(target.FileHash + target.WalletAddress); ok {
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
		task.DownloadTaskMap.Delete(fileHash + setting.WalletAddress)
		task.CleanDownloadFileAndConnMap(fileHash)
	} else {
		notLogin(w)
	}
}

// DownloadSliceCancel
func DownloadSliceCancel(fileHash, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		storeResponseWriter(reqID, w)
		task.DownloadTaskMap.Delete(fileHash + setting.WalletAddress)
		task.CleanDownloadFileAndConnMap(fileHash)
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

func verifySignature(target *protos.ReqDownloadSlice, rsp *protos.RspDownloadSlice) bool {
	val, ok := setting.SPMap.Load(target.SpP2PAddress)
	if !ok {
		utils.ErrorLog("cannot find sp info by given the SP address ", target.SpP2PAddress)
		return false
	}

	spInfo, ok := val.(setting.SPBaseInfo)
	if !ok {
		utils.ErrorLog("Fail to parse SP info ", target.SpP2PAddress)
		return false
	}

	_, pubKeyRaw, err := bech32.DecodeAndConvert(spInfo.P2PPublicKey)
	if err != nil {
		utils.ErrorLog("Error when trying to decode P2P pubKey bech32", err)
		return false
	}

	p2pPubKey := tmed25519.PubKeyEd25519{}
	err = relay.Cdc.UnmarshalBinaryBare(pubKeyRaw, &p2pPubKey)

	if err != nil {
		utils.ErrorLog("Error when trying to read P2P pubKey ed25519 binary", err)
		return false
	}

	if !ed25519crypto.Verify(p2pPubKey[:], []byte(target.P2PAddress+target.FileHash), target.Sign) {
		return false
	}

	return target.SliceInfo.SliceHash == utils.CalcSliceHash(rsp.Data, target.FileHash, target.SliceNumber)
}
