package event

// Author j cc
import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"net/http"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/encryption"
	"github.com/stratosnet/sds/utils/encryption/hdkey"
)

var bpChan = make(chan *msg.RelayMsgBuf, 100)

// ReqDownloadSlice download slice P-PP-storagePP
func ReqDownloadSlice(ctx context.Context, conn core.WriteCloser) {
	utils.Log("ReqDownloadSlice", conn)
	var target protos.ReqDownloadSlice
	if types.UnmarshalData(ctx, &target) {
		// PP will go to DownloadTaskMap to check if has transfer task for this msg, if not, means this PP is the storage PP
		if dlTask, ok := task.DownloadTaskMap.Load(target.FileHash + target.WalletAddress); ok {

			client.DownloadConnMap.Store(target.P2PAddress+target.FileHash, conn)
			downloadTask := dlTask.(*task.DownloadTask)
			if sInfo, ok := downloadTask.SliceInfo[target.SliceInfo.SliceHash]; ok {
				// get all info for the slice
				if sInfo.StoragePpInfo.NetworkAddress == setting.NetworkAddress {
					utils.DebugLog("self is storagePP")
					rsp := types.RspDownloadSliceData(&target)
					if rsp.SliceSize > 0 {
						SendReportDownloadResult(rsp, true)
						splitSendDownloadSliceData(rsp, conn)
					} else {
						downloadWrong(target.TaskId, target.SliceInfo.SliceHash, target.P2PAddress, target.WalletAddress, protos.DownloadWrongType_LOSESLICE)
					}
				} else {
					utils.DebugLog("passagePP received downloadslice request, transfer to :", sInfo.StoragePpInfo.NetworkAddress)
					// transferSendMessageToPPServ(sInfo.StoragePpInfo.NetworkAddress, core.MessageFromContext(ctx))
					if c, ok := client.DownloadPassageway.Load(target.P2PAddress + target.SliceInfo.SliceHash); ok {
						conn := c.(*cf.ClientConn)
						conn.Write(core.MessageFromContext(ctx))
					} else {
						conn := client.NewClient(sInfo.StoragePpInfo.NetworkAddress, false)
						conn.Write(core.MessageFromContext(ctx))
						client.DownloadPassageway.Store(target.P2PAddress+target.SliceInfo.SliceHash, conn)
					}
				}
			} else {

				utils.ErrorLog("download task failed，can't find the slice, fileHash:", target.FileHash, "sliceHash", target.SliceInfo.SliceHash)
			}
		} else {
			utils.DebugLog("storagePP received ReqDownloadSlice,send data to PP ")
			rsp := types.RspDownloadSliceData(&target)
			splitSendDownloadSliceData(rsp, conn)
			if rsp.SliceSize > 0 {
				select {
				//TODO: Change BP to SP
				case bpChan <- types.ReqReportTaskBPData(target.TaskId, uint64(len(rsp.Data))):
					utils.DebugLog("reqReportTaskBPDatareqReportTaskBPDatareqReportTaskBPData")
					// sendBPMessage(bpChan)
				default:
					break
				}
			}
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
			peers.SendMessage(conn, types.RspDownloadSliceDataSplit(rsp, dataStart, dataEnd, offsetStart, offsetEnd,
				rsp.SliceInfo.SliceOffset.SliceOffsetStart, rsp.SliceInfo.SliceOffset.SliceOffsetEnd, false), header.RspDownloadSlice)
			dataStart += setting.MAXDATA
			dataEnd += setting.MAXDATA
			offsetStart += setting.MAXDATA
			offsetEnd += setting.MAXDATA
		} else {
			peers.SendMessage(conn, types.RspDownloadSliceDataSplit(rsp, dataStart, 0, offsetStart, 0,
				rsp.SliceInfo.SliceOffset.SliceOffsetStart, rsp.SliceInfo.SliceOffset.SliceOffsetEnd, true), header.RspDownloadSlice)
			return
		}
	}
}

// RspDownloadSlice storagePP-PP-P
func RspDownloadSlice(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get RspDownloadSlice")
	var target protos.RspDownloadSlice
	if types.UnmarshalData(ctx, &target) {
		if target.P2PAddress != setting.P2PAddress {
			// check if task still exist
			if _, ok := task.DownloadTaskMap.Load(target.FileHash + target.WalletAddress); ok {
				if target.SliceSize > 0 {
					// transfer to
					utils.Log("get RspDownloadSlice transfer to", target.P2PAddress)
					if c, ok := client.DownloadConnMap.Load(target.P2PAddress + target.FileHash); ok {
						conn := c.(*core.ServerConn)
						conn.Write(core.MessageFromContext(ctx))
					} else {
						peers.TransferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
					}
					if target.NeedReport {
						utils.DebugLog("arget.NeedReportarget.NeedReportarget.NeedReportarget.NeedReport")
						SendReportDownloadResult(&target, true)
					}
				} else {
					downloadWrong(target.TaskId, target.SliceInfo.SliceHash, target.P2PAddress, target.WalletAddress, protos.DownloadWrongType_LOSESLICE)
				}
			} else {
				utils.DebugLog("current task is stopped！！！！！！！！！！！！！！！！！！！！！！！！！！")
			}
		} else {
			if f, ok := task.DownloadFileMap.Load(target.FileHash); ok {
				fInfo := f.(*protos.RspFileStorageInfo)
				utils.DebugLog("get a slice -------")
				utils.DebugLog("SliceHash", target.SliceInfo.SliceHash)
				utils.DebugLog("SliceOffset", target.SliceInfo.SliceOffset)
				utils.DebugLog("length", len(target.Data))
				utils.DebugLog("sliceSize", target.SliceSize)
				if !fInfo.IsVideoStream {
					task.DownloadProgress(target.FileHash, uint64(len(target.Data)))
				}
				receiveSliceAndProgress(&target, fInfo)
			}
		}
	}
}

func receiveSliceAndProgress(target *protos.RspDownloadSlice, fInfo *protos.RspFileStorageInfo) {
	if fInfo.EncryptionTag != "" {
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
		return
	}

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

func receivedSlice(target *protos.RspDownloadSlice, fInfo *protos.RspFileStorageInfo) {
	file.SaveDownloadProgress(target.SliceInfo.SliceHash, fInfo.FileName, target.FileHash, target.SavePath)
	target.Result = &protos.Result{
		State: protos.ResultState_RES_SUCCESS,
	}
	if fInfo.IsVideoStream {
		putData(target.ReqId, HTTPDownloadSlice, target)
	}
	SendReportDownloadResult(target, false)
}

// ReportDownloadResult  P-SP OR PP-SP
func SendReportDownloadResult(target *protos.RspDownloadSlice, isPP bool) {
	utils.DebugLog("ReportDownloadResult report result target.FileHash = ", target.FileHash)
	peers.SendMessageToSPServer(types.ReqReportDownloadResultData(target, isPP), header.ReqReportDownloadResult)
	select {
	//TODO: Change BP to SP
	case bpChan <- types.ReqReportTaskBPData(target.TaskId, uint64(target.SliceSize)):
		utils.DebugLog("reqReportTaskBPDatareqReportTaskBPDatareqReportTaskBPData")
		//sendBPMessage(bpChan)
	default:
		break
	}

	task.CleanDownloadTask(target.FileHash, target.SliceInfo.SliceHash, target.P2PAddress, target.WalletAddress)
	// downloadPassageway.Delete(target.WalletAddress + target.SliceInfo.SliceHash)
}

// ReportDownloadResult  P-SP OR PP-SP
func SendReportStreamingResult(target *protos.RspDownloadSlice, isPP bool) {
	peers.SendMessageToSPServer(types.ReqReportDownloadResultData(target, isPP), header.ReqReportDownloadResult)
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
				req := types.ReqDownloadSliceData(target, rsp)
				SendReqDownloadSlice(target, req)
			}
		}
	} else {
		fmt.Println("file exist already!")
	}
}

func SendReqDownloadSlice(target *protos.RspFileStorageInfo, req *protos.ReqDownloadSlice) {
	utils.DebugLog("req = ", req)
	if c, ok := client.PDownloadPassageway.Load(target.FileHash); ok {
		conn := c.(*cf.ClientConn)
		peers.SendMessage(conn, req, header.ReqDownloadSlice)
		utils.DebugLog("DDDDDDDDDDDDDD", conn)
		utils.DebugLog("RRRRRRRRRRRR", client.PPConn)

	} else {
		conn := client.NewClient(client.PPConn.GetName(), false)
		peers.SendMessage(conn, req, header.ReqDownloadSlice)
		client.PDownloadPassageway.Store((target.FileHash), conn)
		utils.DebugLog("WWWWWWWWWWWWWWWWWW", conn)
		utils.DebugLog("ccccccccccccccc", client.PPConn)
	}
}

// RspReportDownloadResult  SP-P OR SP-PP
func RspReportDownloadResult(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get RspReportDownloadResult")
	var target protos.RspReportDownloadResult
	if types.UnmarshalData(ctx, &target) {
		utils.DebugLog("result", target.Result.State, target.Result.Msg)
	}
}

// RspDownloadSliceWrong
func RspDownloadSliceWrong(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("RspDownloadSlice")
	var target protos.RspDownloadSliceWrong
	if types.UnmarshalData(ctx, &target) {
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			utils.DebugLog("RspDownloadSliceWrongRspDownloadSliceWrongRspDownloadSliceWrong", target.NewSliceInfo.SliceStorageInfo.SliceHash)
			if dlTask, ok := task.DownloadTaskMap.Load(target.FileHash + target.WalletAddress); ok {
				downloadTask := dlTask.(*task.DownloadTask)
				if sInfo, ok := downloadTask.SliceInfo[target.NewSliceInfo.SliceStorageInfo.SliceHash]; ok {
					sInfo.StoragePpInfo.P2PAddress = target.NewSliceInfo.StoragePpInfo.P2PAddress
					sInfo.StoragePpInfo.WalletAddress = target.NewSliceInfo.StoragePpInfo.WalletAddress
					sInfo.StoragePpInfo.NetworkAddress = target.NewSliceInfo.StoragePpInfo.NetworkAddress
					peers.TransferSendMessageToPPServ(target.NewSliceInfo.StoragePpInfo.NetworkAddress, types.RspDownloadSliceWrong(&target))
				}
			}
		}
	}
}

func downloadWrong(taskID, sliceHash, p2pAddress, walletAddress string, wrongType protos.DownloadWrongType) {
	utils.DebugLog("downloadWrong, sliceHash: ", sliceHash)
	peers.SendMessageToSPServer(types.ReqDownloadSliceWrong(taskID, sliceHash, p2pAddress, walletAddress, wrongType), header.ReqDownloadSliceWrong)
}

// DownloadSlicePause
func DownloadSlicePause(fileHash, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		peers.SendMessage(client.PPConn, types.ReqDownloadSlicePause(fileHash, reqID), header.ReqDownloadSlicePause)
		// storeResponseWriter(reqID, w)
		task.PCleanDownloadTask(fileHash)
		if c, ok := client.PDownloadPassageway.Load(fileHash); ok {
			conn := c.(*cf.ClientConn)
			conn.ClientClose()
		}
	} else {
		notLogin(w)
	}
}

// DownloadSliceCancel
func DownloadSliceCancel(fileHash, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		peers.SendMessage(client.PPConn, types.ReqDownloadSlicePause(fileHash, reqID), header.ReqDownloadSlicePause)
		storeResponseWriter(reqID, w)
		task.PCleanDownloadTask(fileHash)
		task.PCancelDownloadTask(fileHash)
		if c, ok := client.PDownloadPassageway.Load(fileHash); ok {
			conn := c.(*cf.ClientConn)
			conn.ClientClose()
		}

	} else {
		notLogin(w)
	}
}

// ReqDownloadSlicePause ReqDownloadSlicePause
func ReqDownloadSlicePause(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("ReqDownloadSlicePause&*************************************** ")
	var target protos.ReqDownloadSlicePause
	if types.UnmarshalData(ctx, &target) {
		peers.TransferSendMessageToClient(target.P2PAddress, types.RspDownloadSlicePauseData(&target))
		//
		if dlTask, ok := task.DownloadTaskMap.Load(target.FileHash + target.WalletAddress); ok {
			downloadTask := dlTask.(*task.DownloadTask)
			for k := range downloadTask.SliceInfo {
				key := target.P2PAddress + k
				if c, ok := client.DownloadPassageway.Load(key); ok {
					conn := c.(*cf.ClientConn)
					conn.ClientClose()
				}
				client.DownloadPassageway.Delete(key)
			}
		}
		task.DownloadTaskMap.Delete(target.FileHash + target.WalletAddress)
	}
}

// RspDownloadSlicePause RspDownloadSlicePause
func RspDownloadSlicePause(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspDownloadSlicePause
	if types.UnmarshalData(ctx, &target) {
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			utils.DebugLog("pause successfully, fileHash", target.FileHash)
		} else {
			utils.DebugLog("pause failed, fileHash", target.FileHash, target.Result.Msg)
		}
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
