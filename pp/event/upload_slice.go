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
		peers.SendMessage(conn, requests.UploadSpeedOfProgressData(target.FileHash, uint64(len(target.Data))), header.UploadSpeedOfProgress)
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
			if utils.CalcSliceHash(file.GetSliceData(target.SliceInfo.SliceHash), target.FileHash, target.SliceNumAddr.SliceNumber) == target.SliceInfo.SliceHash {
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

// RspUploadFileSlice
func RspUploadFileSlice(ctx context.Context, conn core.WriteCloser) {
	//check whether self is the target, if not, transfer
	utils.DebugLog("get RspUploadFileSlice")
	var target protos.RspUploadFileSlice
	if requests.UnmarshalData(ctx, &target) {
		utils.DebugLog("P get resp upload slice success sliceNumber", target.SliceNumAddr.SliceNumber, "target.fileHash", target.FileHash)
		utils.DebugLog("target size =", target.SliceSize)
		utils.DebugLog("******************************************")
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			utils.DebugLog("reqReportUploadSliceResultData RspUploadFileSlice")
			peers.SendMessageToSPServer(requests.ReqReportUploadSliceResultData(&target), header.ReqReportUploadSliceResult)
		} else {
			utils.DebugLog("RspUploadFileSlice ErrorLog")
			utils.ErrorLog(target.Result.Msg)
		}
		uploadKeep(target.FileHash, target.TaskId)
	} else {
		utils.ErrorLog("unmarshalData(ctx, &target) error")
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
	fileHash := tk.FileHash
	storageP2pAddress := tk.SliceNumAddr.PpInfo.P2PAddress
	storageNetworkAddress := tk.SliceNumAddr.PpInfo.NetworkAddress
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
				SpP2pAddress: tk.SpP2pAddress,
			}
			utils.DebugLog("*****************", newTask.SliceTotalSize)
			if dataEnd < (tkDataLen + 1) {
				newTask.Data = tk.Data[dataStart:dataEnd]
				utils.DebugLog("dataStart = ", dataStart)
				utils.DebugLog("dataEnd = ", dataEnd)
				sendSlice(requests.ReqUploadFileSliceData(newTask), fileHash, storageP2pAddress, storageNetworkAddress)
				dataStart += setting.MAXDATA
				dataEnd += setting.MAXDATA
			} else {
				utils.DebugLog("dataStart = ", dataStart)
				newTask.Data = tk.Data[dataStart:]
				sendSlice(requests.ReqUploadFileSliceData(newTask), fileHash, storageP2pAddress, storageNetworkAddress)
				return
			}
		}
	} else {
		tk.SliceOffsetInfo.SliceOffset.SliceOffsetStart = 0
		sendSlice(requests.ReqUploadFileSliceData(tk), fileHash, storageP2pAddress, storageNetworkAddress)
	}
}

func sendSlice(pb proto.Message, fileHash, p2pAddress, networkAddress string) {
	utils.DebugLog("sendSlice(pb proto.Message, fileHash, p2pAddress, networkAddress string)",
		fileHash, p2pAddress, networkAddress)

	key := fileHash + p2pAddress

	if c, ok := client.UpConnMap.Load(key); ok {
		conn := c.(*cf.ClientConn)
		err := peers.SendMessage(conn, pb, header.ReqUploadFileSlice)
		if err == nil {
			utils.DebugLog("SendMessage(conn, pb, header.ReqUploadFileSlice) ", conn)
			return
		}
	}

	conn := client.NewClient(networkAddress, false)
	if conn == nil {
		utils.ErrorLog("Fail to create connection with " + networkAddress)
		return
	}

	err := peers.SendMessage(conn, pb, header.ReqUploadFileSlice)
	if err == nil {
		utils.DebugLog("SendMessage(conn, pb, header.ReqUploadFileSlice) ", conn)
		client.UpConnMap.Store(key, conn)
	} else {
		utils.ErrorLog("Fail to send upload slice request to" + networkAddress)
	}
}

// UploadSpeedOfProgress UploadSpeedOfProgress
func UploadSpeedOfProgress(ctx context.Context, conn core.WriteCloser) {

	var target protos.UploadSpeedOfProgress
	if requests.UnmarshalData(ctx, &target) {
		if prg, ok := task.UploadProgressMap.Load(target.FileHash); ok {
			progress := prg.(*task.UpProgress)
			progress.HasUpload += int64(target.SliceSize)
			p := float32(progress.HasUpload) / float32(progress.Total) * 100
			utils.Log("fileHash：", target.FileHash)
			utils.Logf("uploaded：%.2f %% ", p)
			setting.ShowProgress(p)
			ProgressMap.Store(target.FileHash, p)
			if progress.HasUpload >= progress.Total {
				utils.Log("fileHash：", target.FileHash)
				utils.Log(fmt.Sprintf("uploaded：%.2f %% \n", p))
				task.UploadProgressMap.Delete(target.FileHash)
				task.CleanUpConnMap(target.FileHash)
			}
		} else {
			utils.DebugLog("paused!!")
		}
	}
}
