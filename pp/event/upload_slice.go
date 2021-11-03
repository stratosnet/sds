package event

// Author j
import (
	"context"
	"fmt"
	"sync"

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

	"github.com/golang/protobuf/proto"
)

// ProgressMap required by API
var ProgressMap = &sync.Map{}

// ReqUploadFileSlice
func ReqUploadFileSlice(ctx context.Context, conn core.WriteCloser) {
	//check whether self is the target, if not, transfer
	var target protos.ReqUploadFileSlice
	if requests.UnmarshalData(ctx, &target) {
		// passway PP respond to P about progress
		peers.SendMessage(conn, requests.UploadSpeedOfProgressData(target.FileHash, uint64(len(target.Data))), header.UploadSpeedOfProgress)
		if target.SliceNumAddr.PpInfo.NetworkAddress != setting.NetworkAddress {
			utils.DebugLog("transfer to", target.SliceNumAddr.PpInfo.NetworkAddress)
			peers.TransferSendMessageToPPServ(target.SliceNumAddr.PpInfo.NetworkAddress, core.MessageFromContext(ctx))
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
				if utils.CalcSliceHash(file.GetSliceData(target.SliceInfo.SliceHash), target.FileHash) == target.SliceInfo.SliceHash {
					peers.SendMessage(conn, requests.RspUploadFileSliceData(&target), header.RspUploadFileSlice)
					// report upload result to SP
					peers.SendMessageToSPServer(requests.ReqReportUploadSliceResultDataPP(&target), header.ReqReportUploadSliceResult)
					utils.DebugLog("storage PP report to SP upload task finished: ，", target.SliceInfo.SliceHash)
				} else {
					utils.DebugLog("newly stored sliceHash is not equal to target sliceHash!")
				}
			}
		}
	}
}

// RspUploadFileSlice
func RspUploadFileSlice(ctx context.Context, conn core.WriteCloser) {
	//check whether self is the target, if not, transfer
	utils.DebugLog("get RspUploadFileSlice")
	var target protos.RspUploadFileSlice
	if requests.UnmarshalData(ctx, &target) {
		if target.P2PAddress != setting.P2PAddress {

			utils.DebugLog("PP get resp upload slice success, transfer to WalletAddress = ", target.P2PAddress, "sliceNumber= ", target.SliceNumAddr.SliceNumber)
			peers.TransferSendMessageToClient(target.P2PAddress, core.MessageFromContext(ctx))
		} else {
			// target is self, report to SP if success
			utils.DebugLog("P get resp upload slice success sliceNumber", target.SliceNumAddr.SliceNumber, "target.FileHash", target.FileHash)
			utils.DebugLog("target size =", target.SliceSize)
			utils.DebugLog("******************************************")
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				utils.DebugLog("reqReportUploadSliceResultData RspUploadFileSlice")
				peers.SendMessageToSPServer(requests.ReqReportUploadSliceResultData(&target), header.ReqReportUploadSliceResult)
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
func RspReportUploadSliceResult(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get RspReportUploadSliceResult")
	var target protos.RspReportUploadSliceResult
	if requests.UnmarshalData(ctx, &target) {
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			utils.DebugLog("ResultState_RES_SUCCESS, sliceNumber，storageAddress，walletAddress", target.SliceNumAddr.SliceNumber, target.SliceNumAddr.PpInfo.NetworkAddress, target.SliceNumAddr.PpInfo.P2PAddress)
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
				sendSlice(requests.ReqUploadFileSliceData(newTask), newTask.FileHash)
				dataStart += setting.MAXDATA
				dataEnd += setting.MAXDATA
			} else {
				utils.DebugLog("dataStart = ", dataStart)
				newTask.Data = tk.Data[dataStart:]
				sendSlice(requests.ReqUploadFileSliceData(newTask), newTask.FileHash)
				return
			}
		}
	} else {
		tk.SliceOffsetInfo.SliceOffset.SliceOffsetStart = 0
		sendSlice(requests.ReqUploadFileSliceData(tk), tk.FileHash)
	}
}

func sendSlice(pb proto.Message, fileHash string) {
	utils.DebugLog("sendSlice(pb proto.Message, fileHash string)", fileHash)
	if c, ok := client.UpConnMap.Load(fileHash); ok {
		conn := c.(*cf.ClientConn)
		peers.SendMessage(conn, pb, header.ReqUploadFileSlice)
		utils.DebugLog("SendMessage(conn, pb, header.ReqUploadFileSlice)", conn)
	} else {
		utils.DebugLog("paused!!")
	}
}

// UploadSpeedOfProgress UploadSpeedOfProgress
func UploadSpeedOfProgress(ctx context.Context, conn core.WriteCloser) {

	var target protos.UploadSpeedOfProgress
	if requests.UnmarshalData(ctx, &target) {
		utils.DebugLog("~~~~@@@@@@@@@@@@@@@@@@@@@@@@@@!!!!!!!!!!!!!!!!!!!!!!", target.FileHash)
		if prg, ok := task.UploadProgressMap.Load(target.FileHash); ok {
			progress := prg.(*task.UpProgress)
			progress.HasUpload += int64(target.SliceSize)
			p := float32(progress.HasUpload) / float32(progress.Total) * 100
			fmt.Println("fileHash：", target.FileHash)
			fmt.Printf("uploaded：%.2f %% \n", p)
			setting.ShowProgress(p)
			ProgressMap.Store(target.FileHash, p)
			if progress.HasUpload >= progress.Total {
				utils.Log("fileHash：", target.FileHash)
				utils.Log(fmt.Sprintf("uploaded：%.2f %% \n", p))
				task.UploadProgressMap.Delete(target.FileHash)
				client.UpConnMap.Delete(target.FileHash)
			}
		} else {
			utils.DebugLog("paused!!")
		}
	}
}
