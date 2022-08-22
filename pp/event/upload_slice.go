package event

// Author j
import (
	"context"
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils"
)

// ProgressMap required by API
var ProgressMap = &sync.Map{}

// ReqUploadFileSlice
func ReqUploadFileSlice(ctx context.Context, conn core.WriteCloser) {
	//check whether self is the target, if not, transfer
	var target protos.ReqUploadFileSlice
	if requests.UnmarshalData(ctx, &target) {
		if target.Sign == nil || !verifyUploadSliceSign(&target) {
			rsp := &protos.RspUploadFileSlice{
				Result: &protos.Result{
					State: protos.ResultState_RES_FAIL,
					Msg:   "signature validation failed",
				},
			}
			peers.SendMessage(ctx, conn, rsp, header.RspUploadFileSlice)
			return
		}

		if target.SliceNumAddr.PpInfo.P2PAddress != setting.P2PAddress {
			rsp := &protos.RspUploadFileSlice{
				Result: &protos.Result{
					State: protos.ResultState_RES_FAIL,
					Msg:   "mismatch between p2p address in the request and node p2p address.",
				},
			}
			peers.SendMessage(ctx, conn, rsp, header.RspUploadFileSlice)
			return
		}

		peers.SendMessage(ctx, conn, requests.UploadSpeedOfProgressData(target.FileHash, uint64(len(target.Data))), header.UploadSpeedOfProgress)
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
				peers.SendMessage(ctx, conn, requests.RspUploadFileSliceData(&target), header.RspUploadFileSlice)
				// report upload result to SP
				peers.SendMessageToSPServer(ctx, requests.ReqReportUploadSliceResultDataPP(&target), header.ReqReportUploadSliceResult)
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
	pp.DebugLog(ctx, "get RspUploadFileSlice")
	var target protos.RspUploadFileSlice
	if requests.UnmarshalData(ctx, &target) {
		pp.DebugLog(ctx, "P get resp upload slice success sliceNumber", target.SliceNumAddr.SliceNumber, "target.fileHash", target.FileHash)
		pp.DebugLog(ctx, "target size =", target.SliceSize)
		pp.DebugLog(ctx, "******************************************")
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			pp.DebugLog(ctx, "reqReportUploadSliceResultData RspUploadFileSlice")
			peers.SendMessageToSPServer(ctx, requests.ReqReportUploadSliceResultData(&target), header.ReqReportUploadSliceResult)
		} else {
			pp.DebugLog(ctx, "RspUploadFileSlice ErrorLog")
			pp.ErrorLog(ctx, target.Result.Msg)
		}
		uploadKeep(ctx, target.FileHash, target.TaskId)
	} else {
		pp.ErrorLog(ctx, "unmarshalData(ctx, &target) error")
	}
}

// RspReportUploadSliceResult  SP-P OR SP-PP
func RspReportUploadSliceResult(ctx context.Context, conn core.WriteCloser) {
	pp.DebugLog(ctx, "get RspReportUploadSliceResult")
	var target protos.RspReportUploadSliceResult
	if requests.UnmarshalData(ctx, &target) {
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			pp.DebugLog(ctx, "ResultState_RES_SUCCESS, sliceNumber，storageAddress，walletAddress", target.SliceNumAddr.SliceNumber, target.SliceNumAddr.PpInfo.NetworkAddress, target.SliceNumAddr.PpInfo.P2PAddress)
		} else {
			pp.Log(ctx, "ResultState_RES_FAIL : ", target.Result.Msg)
		}
	}
}

// UploadFileSlice
func UploadFileSlice(ctx context.Context, tk *task.UploadSliceTask, sign []byte) {
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
			pp.DebugLog(ctx, "*****************", newTask.SliceTotalSize)
			if dataEnd < (tkDataLen + 1) {
				newTask.Data = tk.Data[dataStart:dataEnd]
				pp.DebugLog(ctx, "dataStart = ", dataStart)
				pp.DebugLog(ctx, "dataEnd = ", dataEnd)
				sendSlice(ctx, requests.ReqUploadFileSliceData(newTask, sign), fileHash, storageP2pAddress, storageNetworkAddress)
				dataStart += setting.MAXDATA
				dataEnd += setting.MAXDATA
			} else {
				pp.DebugLog(ctx, "dataStart = ", dataStart)
				newTask.Data = tk.Data[dataStart:]
				sendSlice(ctx, requests.ReqUploadFileSliceData(newTask, sign), fileHash, storageP2pAddress, storageNetworkAddress)
				return
			}
		}
	} else {
		tk.SliceOffsetInfo.SliceOffset.SliceOffsetStart = 0
		sendSlice(ctx, requests.ReqUploadFileSliceData(tk, sign), fileHash, storageP2pAddress, storageNetworkAddress)
	}
}

func sendSlice(ctx context.Context, pb proto.Message, fileHash, p2pAddress, networkAddress string) {
	pp.DebugLog(ctx, "sendSlice(pb proto.Message, fileHash, p2pAddress, networkAddress string)",
		fileHash, p2pAddress, networkAddress)

	key := fileHash + p2pAddress

	if c, ok := client.UpConnMap.Load(key); ok {
		conn := c.(*cf.ClientConn)
		err := peers.SendMessage(ctx, conn, pb, header.ReqUploadFileSlice)
		if err == nil {
			pp.DebugLog(ctx, "SendMessage(conn, pb, header.ReqUploadFileSlice) ", conn)
			return
		}
	}

	conn := client.NewClient(networkAddress, false)
	if conn == nil {
		pp.ErrorLog(ctx, "Fail to create connection with "+networkAddress)
		return
	}

	err := peers.SendMessage(ctx, conn, pb, header.ReqUploadFileSlice)
	if err == nil {
		pp.DebugLog(ctx, "SendMessage(conn, pb, header.ReqUploadFileSlice) ", conn)
		client.UpConnMap.Store(key, conn)
	} else {
		pp.ErrorLog(ctx, "Fail to send upload slice request to"+networkAddress)
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
			pp.Log(ctx, "fileHash：", target.FileHash)
			pp.Logf(ctx, "uploaded：%.2f %% ", p)
			setting.ShowProgressWithContext(ctx, p)
			ProgressMap.Store(target.FileHash, p)
			if progress.HasUpload >= progress.Total {
				pp.Log(ctx, "fileHash：", target.FileHash)
				pp.Log(ctx, fmt.Sprintf("uploaded：%.2f %% \n", p))
				task.UploadProgressMap.Delete(target.FileHash)
				task.CleanUpConnMap(target.FileHash)
				ScheduleReqBackupStatus(ctx, target.FileHash)
				if file.IsFileRpcRemote(target.FileHash) {
					file.SetRemoteFileResult(target.FileHash, rpc.Result{Return: rpc.SUCCESS})
				}
			}
		} else {
			pp.DebugLog(ctx, "paused!!")
		}
	}
}

func verifyUploadSliceSign(target *protos.ReqUploadFileSlice) bool {
	return requests.VerifySpSignature(target.SpP2PAddress,
		[]byte(target.P2PAddress+target.FileHash+header.ReqUploadFileSlice), target.Sign)
}
