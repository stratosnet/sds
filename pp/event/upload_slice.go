package event

// Author j
import (
	"context"
	"strconv"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
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

var (
	// ProgressMap required by API
	ProgressMap = &sync.Map{}
	//// Maps to record uploading stats
	ReqIdMap          = &sync.Map{} // K: reqId, V: TaskSlice{tkId+sliceNum, up/down}
	upSendCostTimeMap = &sync.Map{} // K: tkId+sliceNum, V: CostTimeStat{TotalCostTime, MiniSliceToCount}

	upRecvCostTimeMap = &sync.Map{} // K: tkId+sliceNum, V: TotalCostTime
)

type TaskSlice struct {
	TkSliceUID string
	IsUpload   bool
}

type CostTimeStat struct {
	TotalCostTime    int64
	MiniSliceToCount int64
}

func ReqUploadFileSlice(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("ReqUploadFileSlice reqID =========", core.GetReqIdFromContext(ctx))
	costTime := core.GetRecvCostTimeFromContext(ctx)

	var target protos.ReqUploadFileSlice
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

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

	// add up costTime
	totalCostTime := costTime
	tkSlice := target.TaskId + strconv.FormatUint(target.SliceNumAddr.SliceNumber, 10)
	if val, ok := upRecvCostTimeMap.Load(tkSlice); ok {
		totalCostTime += val.(int64)
	}
	upRecvCostTimeMap.Store(tkSlice, totalCostTime)

	peers.SendMessage(ctx, conn, requests.UploadSpeedOfProgressData(target.FileHash, uint64(len(target.Data))), header.UploadSpeedOfProgress)

	if !task.SaveUploadFile(&target) {
		// save failed, not handling yet
		utils.ErrorLog("SaveUploadFile failed")
		return
	}

	utils.DebugLogf("ReqUploadFileSlice saving slice %v  current_size %v  total_size %v", target.SliceInfo.SliceHash, file.GetSliceSize(target.SliceInfo.SliceHash), target.SliceSize)
	if file.GetSliceSize(target.SliceInfo.SliceHash) == int64(target.SliceSize) {
		utils.DebugLog("the slice upload finished", target.SliceInfo.SliceHash)
		// respond to PP in case the size is correct but actually not success
		if utils.CalcSliceHash(file.GetSliceData(target.SliceInfo.SliceHash), target.FileHash, target.SliceNumAddr.SliceNumber) == target.SliceInfo.SliceHash {
			utils.DebugLog("RspUploadFileSliceData reqID =========", core.GetReqIdFromContext(ctx))
			peers.SendMessage(ctx, conn, requests.RspUploadFileSliceData(&target), header.RspUploadFileSlice)
			// report upload result to SP

			_, newCtx := peers.AlterReqId(ctx)
			utils.DebugLog("ReqReportUploadSliceResultDataPP reqID =========", core.GetReqIdFromContext(newCtx))
			peers.SendMessageToSPServer(newCtx, requests.ReqReportUploadSliceResultDataPP(&target, totalCostTime), header.ReqReportUploadSliceResult)
			upRecvCostTimeMap.Delete(tkSlice)
			utils.DebugLog("storage PP report to SP upload task finished: ", target.SliceInfo.SliceHash)
		} else {
			utils.ErrorLog("newly stored sliceHash is not equal to target sliceHash!")
		}
	}
}

func RspUploadFileSlice(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("RspUploadFileSlice reqID =========", core.GetReqIdFromContext(ctx))
	var target protos.RspUploadFileSlice
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	pp.DebugLogf(ctx, "get RspUploadFileSlice for file %v  sliceNumber %v  size %v", target.FileHash, target.SliceNumAddr.SliceNumber, target.SliceSize)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		pp.ErrorLog(ctx, "RspUploadFileSlice failure:", target.Result.Msg)
		return
	}
	tkSlice := target.TaskId + strconv.FormatUint(target.SliceNumAddr.SliceNumber, 10)
	if val, ok := upSendCostTimeMap.Load(tkSlice); ok {
		ctStat := val.(CostTimeStat)
		if ctStat.MiniSliceToCount == 0 && ctStat.TotalCostTime > 0 {
			peers.SendMessageToSPServer(ctx, requests.ReqReportUploadSliceResultData(&target, ctStat.TotalCostTime), header.ReqReportUploadSliceResult)
			upSendCostTimeMap.Delete(tkSlice)
		}
	}
}

// RspUploadSlicesWrong updates the destination of slices for an ongoing upload
func RspUploadSlicesWrong(ctx context.Context, _ core.WriteCloser) {
	var target protos.RspUploadSlicesWrong
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	value, ok := task.UploadFileTaskMap.Load(target.FileHash)
	if !ok {
		pp.ErrorLogf(ctx, "File upload task cannot be found for file %v", target.FileHash)
		return
	}
	uploadTask := value.(*task.UploadFileTask)

	if target.Result.State != protos.ResultState_RES_SUCCESS {
		pp.ErrorLog(ctx, "RspUploadSlicesWrong failure:", target.Result.Msg)
		uploadTask.FatalError = errors.New(target.Result.Msg)
		return
	}

	if len(target.Slices) == 0 {
		pp.ErrorLogf(ctx, "No new slices in RspUploadSlicesWrong for file %v. Cannot update slice destinations")
		return
	}

	uploadTask.UpdateSliceDestinations(target.Slices)
	uploadTask.RetryCount++

	// Start upload for all new destinations
	uploadTask.SignalNewDestinations()
}

// RspReportUploadSliceResult  SP-P OR SP-PP
func RspReportUploadSliceResult(ctx context.Context, conn core.WriteCloser) {
	pp.DebugLog(ctx, "get RspReportUploadSliceResult")
	var target protos.RspReportUploadSliceResult
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	if target.Result.State == protos.ResultState_RES_SUCCESS {
		pp.DebugLog(ctx, "ResultState_RES_SUCCESS, sliceNumber，storageAddress，walletAddress", target.SliceNumAddr.SliceNumber, target.SliceNumAddr.PpInfo.NetworkAddress, target.SliceNumAddr.PpInfo.P2PAddress)
	} else {
		pp.Log(ctx, "ResultState_RES_FAIL : ", target.Result.Msg)
	}
}

func UploadFileSlice(ctx context.Context, tk *task.UploadSliceTask, sign []byte) error {
	utils.DebugLog("reqID =========", core.GetReqIdFromContext(ctx))
	tkDataLen := len(tk.Data)
	fileHash := tk.FileHash
	storageP2pAddress := tk.SliceNumAddr.PpInfo.P2PAddress
	storageNetworkAddress := tk.SliceNumAddr.PpInfo.NetworkAddress

	utils.DebugLog("reqID-"+tk.TaskID+" =========", strconv.FormatInt(core.GetReqIdFromContext(ctx), 10))
	tkSliceUID := tk.TaskID + strconv.FormatUint(tk.SliceNumAddr.SliceNumber, 10)
	tkSlice := TaskSlice{
		TkSliceUID: tkSliceUID,
		IsUpload:   true,
	}
	var ctStat = CostTimeStat{}

	if tkDataLen <= setting.MAXDATA {
		tk.SliceOffsetInfo.SliceOffset.SliceOffsetStart = 0

		genReqId, newCtx := peers.AlterReqId(ctx)
		ReqIdMap.Store(genReqId, tkSlice)
		utils.DebugLogf("ReqIdMap.Store <== %v", tkSlice)
		ctStat.MiniSliceToCount = ctStat.MiniSliceToCount + 1
		upSendCostTimeMap.Store(tkSliceUID, ctStat)
		utils.DebugLogf("upSendPacketWgMap.Store <== K:%v, V:%v]", tkSliceUID, ctStat)
		return sendSlice(newCtx, requests.ReqUploadFileSliceData(tk, sign), fileHash, storageP2pAddress, storageNetworkAddress)
	}

	dataStart := 0
	dataEnd := setting.MAXDATA

	if val, ok := upSendCostTimeMap.Load(tk.TaskID + strconv.FormatUint(tk.SliceNumAddr.SliceNumber, 10)); ok {
		ctStat = val.(CostTimeStat)
	}

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
		genReqId, newCtx := peers.AlterReqId(ctx)
		ReqIdMap.Store(genReqId, tkSlice)
		utils.DebugLogf("ReqIdMap.Store <== %v", tkSlice)
		ctStat.MiniSliceToCount = ctStat.MiniSliceToCount + 1
		upSendCostTimeMap.Store(tkSliceUID, ctStat)
		utils.DebugLogf("upSendPacketWgMap.Store <== K:%v, V:%v]", tkSliceUID, ctStat)
		if dataEnd < (tkDataLen + 1) {
			newTask.Data = tk.Data[dataStart:dataEnd]

			pp.DebugLogf(newCtx, "Uploading slice data %v-%v (total %v)", dataStart, dataEnd, newTask.SliceTotalSize)
			err := sendSlice(newCtx, requests.ReqUploadFileSliceData(newTask, sign), fileHash, storageP2pAddress, storageNetworkAddress)
			if err != nil {
				return err
			}
			dataStart += setting.MAXDATA
			dataEnd += setting.MAXDATA
		} else {
			pp.DebugLogf(newCtx, "Uploading slice data %v-%v (total %v)", dataStart, tkDataLen, newTask.SliceTotalSize)
			newTask.Data = tk.Data[dataStart:]
			return sendSlice(newCtx, requests.ReqUploadFileSliceData(newTask, sign), fileHash, storageP2pAddress, storageNetworkAddress)
		}
	}
}

func sendSlice(ctx context.Context, pb proto.Message, fileHash, p2pAddress, networkAddress string) error {
	pp.DebugLog(ctx, "sendSlice(pb proto.Message, fileHash, p2pAddress, networkAddress string)",
		fileHash, p2pAddress, networkAddress)

	key := fileHash + p2pAddress

	if c, ok := client.UpConnMap.Load(key); ok {
		conn := c.(*cf.ClientConn)
		err := peers.SendMessage(ctx, conn, pb, header.ReqUploadFileSlice)
		if err == nil {
			pp.DebugLog(ctx, "SendMessage(conn, pb, header.ReqUploadFileSlice) ", conn)
			return nil
		}
	}

	conn, err := client.NewClient(networkAddress, false)
	if err != nil {
		return errors.Wrap(err, "Failed to create connection with "+networkAddress)
	}

	err = peers.SendMessage(ctx, conn, pb, header.ReqUploadFileSlice)
	if err == nil {
		pp.DebugLog(ctx, "SendMessage(conn, pb, header.ReqUploadFileSlice) ", conn)
		client.UpConnMap.Store(key, conn)
	} else {
		pp.ErrorLog(ctx, "Fail to send upload slice request to "+networkAddress)
	}
	return err
}

func UploadSpeedOfProgress(ctx context.Context, _ core.WriteCloser) {
	var target protos.UploadSpeedOfProgress
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	prg, ok := task.UploadProgressMap.Load(target.FileHash)
	if !ok {
		pp.DebugLog(ctx, "paused!!")
		return
	}

	progress := prg.(*task.UploadProgress)
	progress.HasUpload += int64(target.SliceSize)
	p := float32(progress.HasUpload) / float32(progress.Total) * 100
	pp.Logf(ctx, "fileHash: %v  uploaded：%.2f %% ", target.FileHash, p)
	setting.ShowProgress(ctx, p)
	ProgressMap.Store(target.FileHash, p)
	if progress.HasUpload >= progress.Total {
		task.UploadProgressMap.Delete(target.FileHash)
		task.CleanUpConnMap(target.FileHash)
		ScheduleReqBackupStatus(ctx, target.FileHash)
		if file.IsFileRpcRemote(target.FileHash) {
			file.SetRemoteFileResult(target.FileHash, rpc.Result{Return: rpc.SUCCESS})
		}
	}
}

func verifyUploadSliceSign(target *protos.ReqUploadFileSlice) bool {
	return requests.VerifySpSignature(target.SpP2PAddress,
		[]byte(target.P2PAddress+target.FileHash+header.ReqUploadFileSlice), target.Sign)
}

func HandleSendPacketCostTime(report core.WritePacketCostTime) {
	reqId := report.ReqId
	costTime := report.CostTime

	// get record by reqId
	if val, ok := ReqIdMap.Load(reqId); ok {
		utils.DebugLogf("get reqId[%v] from ReqIdMap, success", reqId)
		tkSlice := val.(TaskSlice)
		if len(tkSlice.TkSliceUID) > 0 {
			ReqIdMap.Delete(reqId)
		}
		utils.DebugLogf("HandleSendPacketCostTime, reqId=%v, isUpload=%v, newReport.costTime=%v, ", reqId, tkSlice.IsUpload, costTime)
		if tkSlice.IsUpload {
			handleUploadSend(tkSlice, costTime)
		} else {
			handleDownloadSend(tkSlice, costTime)

		}
	}
}

func handleUploadSend(tkSlice TaskSlice, costTime int64) {
	var newCostTimeStat = CostTimeStat{}
	if val, ok := upSendCostTimeMap.Load(tkSlice.TkSliceUID); ok {
		utils.DebugLogf("get TkSliceUID[%v] from upSendCostTimeMap, success", tkSlice.TkSliceUID)
		oriCostTimeStat := val.(CostTimeStat)
		newCostTimeStat.TotalCostTime = costTime + oriCostTimeStat.TotalCostTime
		newCostTimeStat.MiniSliceToCount = oriCostTimeStat.MiniSliceToCount - 1
		// not counting if CostTimeStat not found from upSendCostTimeMap
		if newCostTimeStat.MiniSliceToCount >= 0 {
			upSendCostTimeMap.Store(tkSlice.TkSliceUID, newCostTimeStat)
		}
	}
}
