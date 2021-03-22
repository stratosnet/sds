package event

// Author j
import (
	"context"
	"fmt"
	"github.com/qsnetwork/qsds/framework/client/cf"
	"github.com/qsnetwork/qsds/framework/spbf"
	"github.com/qsnetwork/qsds/msg/header"
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/pp/client"
	"github.com/qsnetwork/qsds/pp/file"
	"github.com/qsnetwork/qsds/pp/setting"
	"github.com/qsnetwork/qsds/pp/task"
	"github.com/qsnetwork/qsds/utils"
	"sync"

	"github.com/golang/protobuf/proto"
)

// ProgressMap required by API
var ProgressMap = &sync.Map{}

// ReqUploadFileSlice
func ReqUploadFileSlice(ctx context.Context, conn spbf.WriteCloser) {
	//check whether self is the target, if not, transfer
	var target protos.ReqUploadFileSlice
	if unmarshalData(ctx, &target) {
		// passway PP respond to P about progress
		sendMessage(conn, uploadSpeedOfProgressData(target.FileHash, uint64(len(target.Data))), header.UploadSpeedOfProgress)
		if target.SliceNumAddr.PpInfo.NetworkAddress != setting.NetworkAddress {
			utils.DebugLog("transfer to", target.SliceNumAddr.PpInfo.NetworkAddress)
			transferSendMessageToPPServ(target.SliceNumAddr.PpInfo.NetworkAddress, spbf.MessageFromContext(ctx))
		} else {
			if !task.SaveUploadFile(&target) {
				// save failed, not handing yet
				utils.ErrorLog("SaveUploadFile failed")
				return
			}
			utils.DebugLog("________________________________________________________________________")
			utils.DebugLog("sHash", target.SliceInfo.SliceHash)
			utils.DebugLog("nowsize", file.GetSliceSize(target.SliceInfo.SliceHash))
			utils.DebugLog("target.SliceTotalSize", target.SliceSize)
			if file.GetSliceSize(target.SliceInfo.SliceHash) == int64(target.SliceSize) {
				utils.DebugLog("the slice upload finished", target.SliceInfo.SliceHash)
				// respond to PP in case the size is correct but actually not success
				if utils.CalcHash(file.GetSliceData(target.SliceInfo.SliceHash)) == target.SliceInfo.SliceHash {
					sendMessage(conn, rspUploadFileSliceData(&target), header.RspUploadFileSlice)
					// report upload result to SP
					SendMessageToSPServer(reqReportUploadSliceResultDataPP(&target), header.ReqReportUploadSliceResult)
					utils.DebugLog("storage PP report to SP upload task finished: ，", target.SliceInfo.SliceHash)
				}
			}
		}
	}
}

// RspUploadFileSlice
func RspUploadFileSlice(ctx context.Context, conn spbf.WriteCloser) {
	//check whether self is the target, if not, transfer
	utils.DebugLog("get RspUploadFileSlice")
	var target protos.RspUploadFileSlice
	if unmarshalData(ctx, &target) {
		if target.WalletAddress != setting.WalletAddress {

			utils.DebugLog("PP get resp upload slice success, transfer to WalletAddress = ", target.WalletAddress, "sliceNumber= ", target.SliceNumAddr.SliceNumber)
			transferSendMessageToClient(target.WalletAddress, spbf.MessageFromContext(ctx))
		} else {
			// target is self, report to SP if success
			utils.DebugLog("P get resp upload slice success sliceNumber", target.SliceNumAddr.SliceNumber, "target.FileHash", target.FileHash)
			utils.DebugLog("traget size =", target.SliceSize)
			utils.DebugLog("******************************************")
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				utils.DebugLog("reqReportUploadSliceResultData RspUploadFileSlice")
				SendMessageToSPServer(reqReportUploadSliceResultData(&target), header.ReqReportUploadSliceResult)
			} else {
				utils.DebugLog("RspUploadFileSlice ErrorLog")
				utils.ErrorLog(target.Result.Msg)
			}
			utils.DebugLog("uploadKeep(target.FileHash, target.TaskId)")
			uploadKeep(target.FileHash, target.TaskId)

		}
	} else {
		utils.DebugLog("unmarshalData(ctx, &target)errrrrrrrrrrr ")
	}
}

// RspReportUploadSliceResult  SP-P OR SP-PP
func RspReportUploadSliceResult(ctx context.Context, conn spbf.WriteCloser) {
	utils.DebugLog("get RspReportUploadSliceResult")
	var target protos.RspReportUploadSliceResult
	if unmarshalData(ctx, &target) {
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			utils.DebugLog("ResultState_RES_SUCCESS, sliceNumber，storageAddress，walletAddress", target.SliceNumAddr.SliceNumber, target.SliceNumAddr.PpInfo.NetworkAddress, target.SliceNumAddr.PpInfo.WalletAddress)
		} else {
			utils.Log("ResultState_RES_FAIL : ", target.Result.Msg)
		}
	}
}

// UploadFileSlice
func UploadFileSlice(tk *task.UploadSliceTask) {
	tkDataLen := len(tk.Data)
	if tkDataLen > setting.MAXDATA {
		dataStart := 0
		dataEnd := setting.MAXDATA
		for {
			newTask := &task.UploadSliceTask{
				TaskID:         tk.TaskID,
				FileHash:       tk.FileHash,
				SliceNumAddr:   tk.SliceNumAddr,
				FileCRC:        tk.FileCRC,
				SliceTotalSize: tk.SliceTotalSize,
				SliceOffsetInfo: &protos.SliceOffsetInfo{
					SliceHash: tk.SliceOffsetInfo.SliceHash,
					SliceOffset: &protos.SliceOffset{
						SliceOffsetStart: uint64(dataStart),
						SliceOffsetEnd:   uint64(dataEnd),
					},
				},
			}
			utils.DebugLog("*****************", newTask.SliceTotalSize)
			if dataEnd < (tkDataLen + 1) {
				newTask.Data = tk.Data[dataStart:dataEnd]
				utils.DebugLog("dataStart = ", dataStart)
				utils.DebugLog("dataEnd = ", dataEnd)
				sendSlice(reqUploadFileSliceData(newTask), newTask.FileHash)
				dataStart += setting.MAXDATA
				dataEnd += setting.MAXDATA
			} else {
				utils.DebugLog("dataStart = ", dataStart)
				newTask.Data = tk.Data[dataStart:]
				sendSlice(reqUploadFileSliceData(newTask), newTask.FileHash)
				return
			}
		}
	} else {
		tk.SliceOffsetInfo.SliceOffset.SliceOffsetStart = 0
		sendSlice(reqUploadFileSliceData(tk), tk.FileHash)
	}
}

func sendSlice(pb proto.Message, fileHash string) {
	utils.DebugLog("sendSlice(pb proto.Message, fileHash string)", fileHash)
	if c, ok := client.UpConnMap.Load(fileHash); ok {
		conn := c.(*cf.ClientConn)
		sendMessage(conn, pb, header.ReqUploadFileSlice)
		utils.DebugLog("sendMessage(conn, pb, header.ReqUploadFileSlice)", conn)
	} else {
		utils.DebugLog("paused!!")
	}
}

// UploadSpeedOfProgress UploadSpeedOfProgress
func UploadSpeedOfProgress(ctx context.Context, conn spbf.WriteCloser) {

	var target protos.UploadSpeedOfProgress
	if unmarshalData(ctx, &target) {
		utils.DebugLog("~~~~@@@@@@@@@@@@@@@@@@@@@@@@@@!!!!!!!!!!!!!!!!!!!!!!", target.FileHash)
		if prg, ok := task.UpLoadProgressMap.Load(target.FileHash); ok {
			progress := prg.(*task.UpProgress)
			progress.HasUpload += int64(target.SliceSize)
			p := float32(progress.HasUpload) / float32(progress.Total) * 100
			fmt.Println("fileHash：", target.FileHash)
			fmt.Printf("uploaded：%.2f %% \n", p)
			setting.ShowProgress(p)
			ProgressMap.Store(target.FileHash, p)
			if progress.HasUpload >= progress.Total {
				fmt.Println("file upload finished")
				fmt.Println("fileHash：", target.FileHash)
				task.UpLoadProgressMap.Delete(target.FileHash)
				client.UpConnMap.Delete(target.FileHash)
			}
		} else {
			utils.DebugLog("paused!!")
		}
	}
}
