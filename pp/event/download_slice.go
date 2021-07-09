package event

// Author j cc
import (
	"context"
	"fmt"
	"net/http"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils"
)

var bpChan = make(chan *msg.RelayMsgBuf, 100)

// ReqDownloadSlice download slice P-PP-storagePP
func ReqDownloadSlice(ctx context.Context, conn spbf.WriteCloser) {
	utils.Log("ReqDownloadSlice", conn)
	var target protos.ReqDownloadSlice
	if unmarshalData(ctx, &target) {
		// PP will go to DownloadTaskMap to check if has transfer task for this msg, if not, means this PP is the storage PP
		if dlTask, ok := task.DownloadTaskMap.Load(target.FileHash + target.WalletAddress); ok {

			client.DownloadConnMap.Store(target.P2PAddress+target.FileHash, conn)
			downloadTask := dlTask.(*task.DownloadTask)
			if sInfo, ok := downloadTask.SliceInfo[target.SliceInfo.SliceHash]; ok {
				// get all info for the slice
				if sInfo.StoragePpInfo.NetworkAddress == setting.NetworkAddress {
					utils.DebugLog("self is storagePP")
					rsp := rspDownloadSliceData(&target)
					if rsp.SliceSize > 0 {
						sendReportDownloadResult(rsp, true)
						splitSendDownloadSliceData(rsp, conn)
					} else {
						downloadWrong(target.TaskId, target.SliceInfo.SliceHash, target.P2PAddress, target.WalletAddress, protos.DownloadWrongType_LOSESLICE)
					}
				} else {
					utils.DebugLog("passagePP received downloadslice reqest, transfer to :", sInfo.StoragePpInfo.NetworkAddress)
					// transferSendMessageToPPServ(sInfo.StoragePpInfo.NetworkAddress, spbf.MessageFromContext(ctx))
					if c, ok := client.DownloadPassageway.Load(target.P2PAddress + target.SliceInfo.SliceHash); ok {
						conn := c.(*cf.ClientConn)
						conn.Write(spbf.MessageFromContext(ctx))
					} else {
						conn := client.NewClient(sInfo.StoragePpInfo.NetworkAddress, false)
						conn.Write(spbf.MessageFromContext(ctx))
						client.DownloadPassageway.Store(target.P2PAddress+target.SliceInfo.SliceHash, conn)
					}
				}
			} else {

				utils.ErrorLog("download task failed，can't find the slice, fileHash:", target.FileHash, "sliceHash", target.SliceInfo.SliceHash)
			}
		} else {
			utils.DebugLog("storagePP received ReqDownloadSlice,send data to PP ")
			rsp := rspDownloadSliceData(&target)
			splitSendDownloadSliceData(rsp, conn)
			if rsp.SliceSize > 0 {
				select {
				//TODO: Change BP to SP
				case bpChan <- reqReportTaskBPData(target.TaskId, uint64(len(rsp.Data))):
					utils.DebugLog("reqReportTaskBPDatareqReportTaskBPDatareqReportTaskBPData")
					// sendBPMessage(bpChan)
				default:
					break
				}
			}
		}
	}
}

func splitSendDownloadSliceData(rsp *protos.RspDownloadSlice, conn spbf.WriteCloser) {
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
			sendMessage(conn, rspDownloadSliceDataSplit(rsp, dataStart, dataEnd, offsetStart, offsetEnd, false), header.RspDownloadSlice)
			dataStart += setting.MAXDATA
			dataEnd += setting.MAXDATA
			offsetStart += setting.MAXDATA
			offsetEnd += setting.MAXDATA
		} else {
			sendMessage(conn, rspDownloadSliceDataSplit(rsp, dataStart, 0, offsetStart, rsp.SliceInfo.SliceOffset.SliceOffsetEnd, true), header.RspDownloadSlice)
			return
		}
	}
}

// RspDownloadSlice storagePP-PP-P
func RspDownloadSlice(ctx context.Context, conn spbf.WriteCloser) {
	utils.DebugLog("get RspDownloadSlice")
	var target protos.RspDownloadSlice
	if unmarshalData(ctx, &target) {
		if target.P2PAddress != setting.P2PAddress {
			// check if task still exist
			if _, ok := task.DownloadTaskMap.Load(target.FileHash + target.WalletAddress); ok {
				if target.SliceSize > 0 {
					// transfer to
					utils.Log("get RspDownloadSlice transfer to", target.P2PAddress)
					if c, ok := client.DownloadConnMap.Load(target.P2PAddress + target.FileHash); ok {
						conn := c.(*spbf.ServerConn)
						conn.Write(spbf.MessageFromContext(ctx))
					} else {
						transferSendMessageToClient(target.P2PAddress, spbf.MessageFromContext(ctx))
					}
					if target.NeedReport {
						utils.DebugLog("arget.NeedReportarget.NeedReportarget.NeedReportarget.NeedReport")
						sendReportDownloadResult(&target, true)
					}
				} else {
					downloadWrong(target.TaskId, target.SliceInfo.SliceHash, target.P2PAddress, target.WalletAddress, protos.DownloadWrongType_LOSESLICE)
				}
			} else {
				utils.DebugLog("current task is stopped！！！！！！！！！！！！！！！！！！！！！！！！！！")
			}
		} else {
			if _, ok := task.DownloadFileMap.Load(target.FileHash); ok {
				utils.DebugLog("get a slice -------")
				utils.DebugLog("SliceHash", target.SliceInfo.SliceHash)
				utils.DebugLog("SliceOffset", target.SliceInfo.SliceOffset)
				utils.DebugLog("length", len(target.Data))
				utils.DebugLog("sliceSize", target.SliceSize)
				task.DownloadProgress(target.FileHash, uint64(len(target.Data)))

				if task.SaveDownloadFile(&target) {
					dataLen := uint64(len(target.Data))
					if s, ok := task.DownloadSliceProgress.Load(target.SliceInfo.SliceHash); ok {
						alreadySize := s.(uint64)
						alreadySize += dataLen
						if alreadySize == target.SliceSize {
							utils.DebugLog("slice download finished", target.SliceInfo.SliceHash)
							task.DownloadSliceProgress.Delete(target.SliceInfo.SliceHash)
							receivedSlice(&target)
						} else {
							task.DownloadSliceProgress.Store(target.SliceInfo.SliceHash, alreadySize)
						}
					} else {
						// if data is sent at once
						if target.SliceSize == dataLen {
							receivedSlice(&target)
						} else {
							task.DownloadSliceProgress.Store(target.SliceInfo.SliceHash, dataLen)
						}
					}
				}
			}
		}
	}
}

func receivedSlice(target *protos.RspDownloadSlice) {
	if f, ok := task.DownloadFileMap.Load(target.FileHash); ok {
		fInfo := f.(*protos.RspFileStorageInfo)
		file.SaveDownloadProgress(target.SliceInfo.SliceHash, fInfo.FileName, target.FileHash, target.SavePath)
	}
	sendReportDownloadResult(target, false)
}

// ReportDownloadResult  P-SP OR PP-SP
func sendReportDownloadResult(target *protos.RspDownloadSlice, isPP bool) {
	utils.DebugLog("ReportDownloadResult report result target.FileHash = ", target.FileHash)
	SendMessageToSPServer(reqReportDownloadResultData(target, isPP), header.ReqReportDownloadResult)
	select {
	//TODO: Change BP to SP
	case bpChan <- reqReportTaskBPData(target.TaskId, uint64(target.SliceSize)):
		utils.DebugLog("reqReportTaskBPDatareqReportTaskBPDatareqReportTaskBPData")
		//sendBPMessage(bpChan)
	default:
		break
	}

	task.CleanDownloadTask(target.FileHash, target.SliceInfo.SliceHash, target.P2PAddress, target.WalletAddress)
	// downloadPassageway.Delete(target.WalletAddress + target.SliceInfo.SliceHash)
}

// DownloadFileSlice
func DownloadFileSlice(target *protos.RspFileStorageInfo) {
	utils.DebugLog("file size: ", target.FileSize)
	sp := &task.DownloadSP{
		TotalSize:    int64(target.FileSize),
		DownloadSize: 0,
	}
	task.DownloadSpeedOfProgress.Store(target.FileHash, sp)
	if !file.CheckFileExisting(target.FileHash, target.FileName, target.SavePath) {
		for _, rsp := range target.SliceInfo {

			utils.DebugLog("taskid ======= ", rsp.TaskId)
			if file.CheckSliceExisting(target.FileHash, target.FileName, rsp.SliceStorageInfo.SliceHash, target.SavePath) {
				utils.Log("slice exist already,", rsp.SliceStorageInfo.SliceHash)
				task.DownloadProgress(target.FileHash, rsp.SliceStorageInfo.SliceSize)
			} else {
				utils.DebugLog("request download data")
				req := reqDownloadSliceData(target, rsp)
				utils.DebugLog("req = ", req)
				if c, ok := client.PDownloadPassageway.Load(target.FileHash); ok {
					conn := c.(*cf.ClientConn)
					sendMessage(conn, req, header.ReqDownloadSlice)
					utils.DebugLog("DDDDDDDDDDDDDD", conn)
					utils.DebugLog("RRRRRRRRRRRR", client.PPConn)

				} else {
					conn := client.NewClient(client.PPConn.GetName(), false)
					sendMessage(conn, req, header.ReqDownloadSlice)
					client.PDownloadPassageway.Store((target.FileHash), conn)
					utils.DebugLog("WWWWWWWWWWWWWWWWWW", conn)
					utils.DebugLog("ccccccccccccccc", client.PPConn)
				}
			}
		}
	} else {
		fmt.Println("file exist already!")
	}
}

// RspReportDownloadResult  SP-P OR SP-PP
func RspReportDownloadResult(ctx context.Context, conn spbf.WriteCloser) {
	utils.DebugLog("get RspReportDownloadResult")
	var target protos.RspReportDownloadResult
	if unmarshalData(ctx, &target) {
		utils.DebugLog("result", target.Result.State, target.Result.Msg)
	}
}

// RspDownloadSliceWrong
func RspDownloadSliceWrong(ctx context.Context, conn spbf.WriteCloser) {
	utils.DebugLog("RspDownloadSlice")
	var target protos.RspDownloadSliceWrong
	if unmarshalData(ctx, &target) {
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			utils.DebugLog("RspDownloadSliceWrongRspDownloadSliceWrongRspDownloadSliceWrong", target.NewSliceInfo.SliceStorageInfo.SliceHash)
			if dlTask, ok := task.DownloadTaskMap.Load(target.FileHash + target.WalletAddress); ok {
				downloadTask := dlTask.(*task.DownloadTask)
				if sInfo, ok := downloadTask.SliceInfo[target.NewSliceInfo.SliceStorageInfo.SliceHash]; ok {
					sInfo.StoragePpInfo.P2PAddress = target.NewSliceInfo.StoragePpInfo.P2PAddress
					sInfo.StoragePpInfo.WalletAddress = target.NewSliceInfo.StoragePpInfo.WalletAddress
					sInfo.StoragePpInfo.NetworkAddress = target.NewSliceInfo.StoragePpInfo.NetworkAddress
					transferSendMessageToPPServ(target.NewSliceInfo.StoragePpInfo.NetworkAddress, rspDownloadSliceWrong(&target))
				}
			}
		}
	}
}

func downloadWrong(taskID, sliceHash, p2pAddress, walletAddress string, wrongType protos.DownloadWrongType) {
	utils.DebugLog("downloadWrong, sliceHash: ", sliceHash)
	SendMessageToSPServer(reqDownloadSliceWrong(taskID, sliceHash, p2pAddress, walletAddress, wrongType), header.ReqDownloadSliceWrong)
}

// DownloadSlicePause
func DownloadSlicePause(fileHash, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		sendMessage(client.PPConn, reqDownloadSlicePause(fileHash, reqID), header.ReqDownloadSlicePause)
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
		sendMessage(client.PPConn, reqDownloadSlicePause(fileHash, reqID), header.ReqDownloadSlicePause)
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
func ReqDownloadSlicePause(ctx context.Context, conn spbf.WriteCloser) {
	utils.DebugLog("ReqDownloadSlicePause&*************************************** ")
	var target protos.ReqDownloadSlicePause
	if unmarshalData(ctx, &target) {
		transferSendMessageToClient(target.P2PAddress, rspDownloadSlicePauseData(&target))
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
func RspDownloadSlicePause(ctx context.Context, conn spbf.WriteCloser) {
	var target protos.RspDownloadSlicePause
	if unmarshalData(ctx, &target) {
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			utils.DebugLog("pause successfully, fileHash", target.FileHash)
		} else {
			utils.DebugLog("pause failed, fileHash", target.FileHash, target.Result.Msg)
		}
	}
}
