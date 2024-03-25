package event

// Author j
import (
	"context"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/crypto"
	"github.com/stratosnet/sds/framework/metrics"
	"github.com/stratosnet/sds/framework/msg/header"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/sds-msg/protos"
)

var (
	mutexHandleSendCostTime = &sync.Mutex{}
	//// Maps to record uploading stats
	PacketIdMap       = &sync.Map{} // K: reqId, V: TaskSlice{tkId+sliceNum, up/down}
	UpSendCostTimeMap = &upSendCostTime{
		dataMap: utils.NewAutoCleanUnsafeMap(30 * time.Minute), // make(map[string]*CostTimeStat) // K: tkId+sliceNum, V: CostTimeStat{TotalCostTime, PacketCount}
		mux:     sync.Mutex{},
	}
	UpRecvCostTimeMap = &upRecvCostTime{
		dataMap: utils.NewAutoCleanUnsafeMap(30 * time.Minute), // make(map[string]int64), // K: tkId+sliceNum, V: TotalCostTime
		mux:     sync.Mutex{},
	}

	uploadSliceSpamCheckMap = utils.NewAutoCleanMap(setting.SpamThresholdSliceOperations)
	backupSliceSpamCheckMap = utils.NewAutoCleanMap(setting.SpamThresholdSliceOperations)
)

const (
	SliceInvalid  = 0
	SliceUpload   = 1
	SliceDownload = 2
	SliceBackup   = 3
	SliceTransfer = 4
)

type upSendCostTime struct {
	dataMap *utils.AutoCleanUnsafeMap //map[string]*CostTimeStat // K: tkId+sliceNum, V: CostTimeStat{TotalCostTime, PacketCount}
	mux     sync.Mutex
}
type upRecvCostTime struct {
	dataMap *utils.AutoCleanUnsafeMap // map[string]int64 // K: tkId+sliceNum, V: TotalCostTime
	mux     sync.Mutex
}

type TaskSlice struct {
	TkSliceUID    string
	SliceType     int32
	fileHash      string
	TaskId        string
	SliceHash     string
	SpP2pAddress  string
	OriginDeleted bool
}

type CostTimeStat struct {
	TotalCostTime int64
	PacketCount   int64
}

func GetOngoingUploadTaskCount() int {
	UpRecvCostTimeMap.mux.Lock()
	count := UpRecvCostTimeMap.dataMap.Len()
	UpRecvCostTimeMap.mux.Unlock()
	return count
}

// ReqUploadFileSlice storage PP receives a request with file data from the PP who initiated uploading
func ReqUploadFileSlice(ctx context.Context, conn core.WriteCloser) {
	costTime := core.GetRecvCostTimeFromContext(ctx)
	var target protos.ReqUploadFileSlice
	if err := VerifyMessage(ctx, header.ReqUploadFileSlice, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}

	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	defer utils.ReleaseBuffer(target.Data)
	rspUploadFile := target.RspUploadFile
	var slice *protos.SliceHashAddr
	for _, slice = range rspUploadFile.Slices {
		if slice.SliceNumber == target.SliceNumber {
			break
		}
	}

	if target.PieceOffset.SliceOffsetStart > slice.SliceSize || target.PieceOffset.SliceOffsetStart%setting.MaxData != 0 {
		rsp := &protos.RspUploadFileSlice{
			Result: &protos.Result{
				State: protos.ResultState_RES_FAIL,
				Msg:   "the offset of the piece in slice is wrong",
			},
		}
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspUploadFileSlice)
		return
	}

	// spam check
	key := rspUploadFile.TaskId + strconv.FormatInt(int64(target.SliceNumber), 10) +
		strconv.FormatInt(int64(target.PieceOffset.SliceOffsetStart), 10) + target.P2PAddress +
		strconv.FormatInt(rspUploadFile.TimeStamp, 10)
	if _, ok := uploadSliceSpamCheckMap.Load(key); ok {
		rsp := &protos.RspUploadFileSlice{
			Result: &protos.Result{
				State: protos.ResultState_RES_FAIL,
				Msg:   "repeated upload slice request, refused",
			},
		}
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspUploadFileSlice)
		return
	} else {
		var a any
		uploadSliceSpamCheckMap.Store(key, a)
	}

	fileHash := rspUploadFile.FileHash
	newSlice := slice
	sliceSizeFromMsg := slice.SliceOffset.SliceOffsetEnd - slice.SliceOffset.SliceOffsetStart

	if slice.PpInfo.P2PAddress != p2pserver.GetP2pServer(ctx).GetP2PAddress().String() {
		rsp := &protos.RspUploadFileSlice{
			Result: &protos.Result{
				State: protos.ResultState_RES_FAIL,
				Msg:   "mismatch between p2p address in the request and node p2p address.",
			},
		}
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspUploadFileSlice)
		return
	}

	// add up costTime
	totalCostTime := costTime
	tkSlice := rspUploadFile.TaskId + strconv.FormatUint(target.SliceNumber, 10)
	UpRecvCostTimeMap.mux.Lock()
	if val, ok := UpRecvCostTimeMap.dataMap.Load(tkSlice); ok {
		totalCostTime += val.(int64)
	}
	UpRecvCostTimeMap.dataMap.Store(tkSlice, totalCostTime)
	UpRecvCostTimeMap.mux.Unlock()
	timeEntry := time.Now().UnixMicro() - core.TimeRcv
	pieceSize := target.PieceOffset.SliceOffsetEnd - target.PieceOffset.SliceOffsetStart
	_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, requests.UploadSpeedOfProgressData(fileHash, pieceSize, (target.SliceNumber-1)*33554432+target.PieceOffset.SliceOffsetStart, timeEntry), header.UploadSpeedOfProgress)

	needToSave := true
	sliceSize, _ := file.GetSliceSize(target.SliceHash)
	if target.PieceOffset.SliceOffsetStart < uint64(sliceSize) {
		if target.PieceOffset.SliceOffsetEnd != sliceSizeFromMsg {
			return
		}
		needToSave = false
	}

	if needToSave {
		if err := task.SaveUploadFile(&target); err != nil {
			// failed saving the packet. It is not handled.
			utils.ErrorLog("SaveUploadFile failed", err.Error())
			return
		}
	}

	// get the size again after saving the packet
	sliceSize, err := file.GetSliceSize(target.SliceHash)
	if err != nil {
		utils.ErrorLog("Failed getting slice size", err.Error())
		return
	}

	// check if the slice has finished
	utils.DebugLogf("ReqUploadFileSlice saving slice %v  current_size %v  total_size %v", target.SliceHash, sliceSize, sliceSizeFromMsg)
	if sliceSize == int64(sliceSizeFromMsg) {
		utils.DebugLog("the slice upload finished", target.SliceHash)
		// respond to PP in case the size is correct but actually not success
		sliceData, err := file.GetSliceData(target.SliceHash)
		if err != nil {
			utils.ErrorLog("Failed getting slice data", err.Error())
			return
		}
		sliceHash, err := crypto.CalcSliceHash(sliceData, fileHash, target.SliceNumber)
		if err != nil {
			utils.ErrorLog("Failed to calc slice hash", err.Error())
			return
		}
		if sliceHash == target.SliceHash {
			_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, requests.RspUploadFileSliceData(ctx, &target), header.RspUploadFileSlice)
			// report upload result to SP
			newSlice.SliceHash = target.SliceHash
			_, newCtx := p2pserver.CreateNewContextPacketId(ctx)
			utils.DebugLog("ReqReportUploadSliceResultDataPP reqID =========", core.GetReqIdFromContext(newCtx))
			reportResultReq := requests.ReqReportUploadSliceResultData(ctx, target.RspUploadFile.TaskId,
				target.RspUploadFile.FileHash,
				target.RspUploadFile.SpP2PAddress,
				target.P2PAddress,
				newSlice,
				totalCostTime)

			p2pserver.GetP2pServer(ctx).SendMessageToSPServer(newCtx, reportResultReq, header.ReqReportUploadSliceResult)
			metrics.StoredSliceCount.WithLabelValues("upload").Inc()
			instantInboundSpeed := float64(sliceSizeFromMsg) / math.Max(float64(totalCostTime), 1)
			metrics.InboundSpeed.WithLabelValues(reportResultReq.OpponentP2PAddress).Set(instantInboundSpeed)
			UpRecvCostTimeMap.mux.Lock()
			UpRecvCostTimeMap.dataMap.Delete(tkSlice)
			UpRecvCostTimeMap.mux.Unlock()
			utils.DebugLog("storage PP report to SP upload task finished: ", target.SliceHash)
		} else {
			utils.ErrorLog("newly stored sliceHash is not equal to target sliceHash!")
			if err = file.DeleteSlice(target.SliceHash); err != nil {
				utils.ErrorLog("failed removing slice file!")
			}
		}
	}
}

func RspUploadFileSlice(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspUploadFileSlice
	if err := VerifyMessage(ctx, header.RspUploadFileSlice, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error(),
			", please make sure your upload bandwidth is enough and retry")
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	if target.Result.State != protos.ResultState_RES_SUCCESS {
		pp.ErrorLog(ctx, "RspUploadFileSlice failure:", target.Result.Msg)
		return
	}
	if target.Slice == nil {
		pp.ErrorLog(ctx, "RspUploadFileSlice failure: no slice included")
		return
	}
	metrics.UploadPerformanceLogNow(target.FileHash + ":RCV_RSP_SLICE:" + strconv.FormatInt(int64(target.Slice.SliceNumber), 10))
	pp.DebugLogf(ctx, "get RspUploadFileSlice for file %v  sliceNumber %v  size %v",
		target.FileHash, target.Slice.SliceNumber, target.Slice.SliceSize)

	tkSlice := target.TaskId + strconv.FormatUint(target.Slice.SliceNumber, 10)
	UpSendCostTimeMap.mux.Lock()
	defer UpSendCostTimeMap.mux.Unlock()
	if val, ok := UpSendCostTimeMap.dataMap.Load(tkSlice); ok {
		ctStat := val.(CostTimeStat)
		utils.DebugLogf("ctStat is %v", ctStat)
		if ctStat.PacketCount == 0 && ctStat.TotalCostTime > 0 {
			value, ok := task.UploadFileTaskMap.Load(target.FileHash)
			if !ok {
				utils.DebugLog("failed finding upload task for file", target.FileHash)
				return
			}
			fileTask := value.(*task.UploadFileTask)
			if err := fileTask.SetUploadSliceStatus(target.SliceHash, task.SLICE_STATUS_FINISHED); err != nil {
				utils.DebugLog("failed setting upload slice status,", err.Error())
				return
			}
			fileTask.Touch()
			p := fileTask.GetUploadProgress()
			pp.Logf(ctx, "fileHash: %v  uploaded：%.2f %% ", target.FileHash, p)
			setting.ShowProgress(ctx, p)

			target.Slice.SliceHash = target.SliceHash
			reportReq := requests.ReqReportUploadSliceResultData(ctx,
				target.TaskId,
				target.FileHash,
				target.SpP2PAddress,
				target.Slice.PpInfo.P2PAddress,
				target.Slice,
				ctStat.TotalCostTime)
			p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, reportReq, header.ReqReportUploadSliceResult)
			instantOutboundSpeed := float64(target.Slice.SliceSize) / math.Max(float64(ctStat.TotalCostTime), 1)
			metrics.OutboundSpeed.WithLabelValues(target.P2PAddress).Set(instantOutboundSpeed)

			UpSendCostTimeMap.dataMap.Delete(tkSlice)
		}
	} else {
		utils.DebugLogf("tkSlice [%v] not found in RspUploadFileSlice", tkSlice)
	}
}

// ReqBackupFileSlice
func ReqBackupFileSlice(ctx context.Context, conn core.WriteCloser) {
	costTime := core.GetRecvCostTimeFromContext(ctx)
	var target protos.ReqBackupFileSlice

	if err := VerifyMessage(ctx, header.ReqBackupFileSlice, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	defer utils.ReleaseBuffer(target.Data)
	// spam check
	key := target.RspBackupFile.TaskId + strconv.FormatInt(int64(target.SliceNumber), 10) + target.P2PAddress +
		strconv.FormatInt(target.RspBackupFile.TimeStamp, 10)
	if _, ok := backupSliceSpamCheckMap.Load(key); ok {
		rsp := &protos.RspBackupFileSlice{
			Result: &protos.Result{
				State: protos.ResultState_RES_FAIL,
				Msg:   "repeated backup slice request, refused",
			},
		}
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspBackupFileSlice)
		return
	} else {
		var a any
		backupSliceSpamCheckMap.Store(key, a)
	}

	// setting local variables
	fileHash := target.RspBackupFile.FileHash
	var slice *protos.SliceHashAddr
	for _, slice = range target.RspBackupFile.Slices {
		if slice.SliceNumber == target.SliceNumber {
			break
		}
	}
	sliceSizeFromMsg := slice.SliceOffset.SliceOffsetEnd - slice.SliceOffset.SliceOffsetStart

	if slice.PpInfo.P2PAddress != p2pserver.GetP2pServer(ctx).GetP2PAddress().String() {
		rsp := &protos.RspBackupFileSlice{
			Result: &protos.Result{
				State: protos.ResultState_RES_FAIL,
				Msg:   "mismatch between p2p address in the request and node p2p address.",
			},
		}
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspBackupFileSlice)
		return
	}

	// add up costTime
	totalCostTime := costTime
	tkSlice := target.RspBackupFile.TaskId + strconv.FormatUint(target.SliceNumber, 10)
	UpRecvCostTimeMap.mux.Lock()
	if val, ok := UpRecvCostTimeMap.dataMap.Load(tkSlice); ok {
		totalCostTime += val.(int64)
	}
	UpRecvCostTimeMap.dataMap.Store(tkSlice, totalCostTime)
	UpRecvCostTimeMap.mux.Unlock()
	timeEntry := time.Now().UnixMicro() - core.TimeRcv

	msg := requests.UploadSpeedOfProgressData(fileHash, uint64(len(target.Data)),
		(target.SliceNumber-1)*setting.MaxSliceSize+target.PieceOffset.SliceOffsetStart, timeEntry)
	_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, msg, header.UploadSpeedOfProgress)
	if err := task.SaveBackuptFile(&target); err != nil {
		// save failed, not handling yet
		utils.ErrorLog("SaveUploadFile failed", err.Error())
		return
	}
	sliceSize, err := file.GetSliceSize(target.SliceHash)
	if err != nil {
		utils.ErrorLog("Failed getting slice size", err.Error())
		return
	}

	// check if the slice has finished
	utils.DebugLogf("ReqUploadFileSlice saving slice %v  current_size %v  total_size %v", target.SliceHash, sliceSize, sliceSizeFromMsg)
	if sliceSize == int64(sliceSizeFromMsg) {
		utils.DebugLog("the slice upload finished", target.SliceHash)
		// respond to PP in case the size is correct but actually not success
		sliceData, err := file.GetSliceData(target.SliceHash)
		if err != nil {
			utils.ErrorLog("Failed getting slice data", err.Error())
			return
		}
		sliceHash, err := crypto.CalcSliceHash(sliceData, fileHash, target.SliceNumber)
		if err != nil {
			utils.ErrorLog("Failed to calc slice hash", err)
			return
		}
		if sliceHash == target.SliceHash {
			_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, requests.RspBackupFileSliceData(&target), header.RspBackupFileSlice)
			// report upload result to SP

			_, newCtx := p2pserver.CreateNewContextPacketId(ctx)
			utils.DebugLog("ReqReportUploadSliceResultDataPP reqID =========", core.GetReqIdFromContext(newCtx))
			reportResultReq := requests.ReqReportUploadSliceResultData(
				ctx,
				target.RspBackupFile.TaskId,
				target.RspBackupFile.FileHash,
				target.RspBackupFile.SpP2PAddress,
				p2pserver.GetP2pServer(ctx).GetP2PAddress().String(),
				slice,
				totalCostTime)
			p2pserver.GetP2pServer(ctx).SendMessageToSPServer(newCtx, reportResultReq, header.ReqReportUploadSliceResult)
			metrics.StoredSliceCount.WithLabelValues("upload").Inc()
			instantInboundSpeed := float64(sliceSizeFromMsg) / math.Max(float64(totalCostTime), 1)
			metrics.InboundSpeed.WithLabelValues(reportResultReq.OpponentP2PAddress).Set(instantInboundSpeed)
			UpRecvCostTimeMap.mux.Lock()
			UpRecvCostTimeMap.dataMap.Delete(tkSlice)
			UpRecvCostTimeMap.mux.Unlock()
			utils.DebugLog("storage PP report to SP upload task finished: ", target.SliceHash)
		} else {
			utils.ErrorLog("newly stored sliceHash is not equal to target sliceHash!")
		}
	}
}

func RspBackupFileSlice(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspBackupFileSlice
	if err := VerifyMessage(ctx, header.RspBackupFileSlice, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	pp.DebugLogf(ctx, "get RspUploadFileSlice for file %v  sliceNumber %v  size %v", target.FileHash, target.Slice.SliceNumber, target.SliceSize)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		pp.ErrorLog(ctx, "RspUploadFileSlice failure:", target.Result.Msg)
		return
	}
	tkSlice := target.TaskId + strconv.FormatUint(target.Slice.SliceNumber, 10)
	UpSendCostTimeMap.mux.Lock()
	defer UpSendCostTimeMap.mux.Unlock()
	if val, ok := UpSendCostTimeMap.dataMap.Load(tkSlice); ok {
		ctStat := val.(CostTimeStat)
		utils.DebugLogf("ctStat is %v", ctStat)
		if ctStat.PacketCount == 0 && ctStat.TotalCostTime > 0 {
			reportReq := requests.ReqReportUploadSliceResultData(
				ctx,
				target.TaskId,
				target.FileHash,
				target.SpP2PAddress,
				p2pserver.GetP2pServer(ctx).GetP2PAddress().String(),
				target.Slice,
				ctStat.TotalCostTime)

			p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, reportReq, header.ReqReportUploadSliceResult)
			instantOutboundSpeed := float64(target.SliceSize) / math.Max(float64(ctStat.TotalCostTime), 1)
			metrics.OutboundSpeed.WithLabelValues(target.P2PAddress).Set(instantOutboundSpeed)

			UpSendCostTimeMap.dataMap.Delete(tkSlice)
		}
	} else {
		utils.DebugLogf("tkSlice [%v] not found in RspUploadFileSlice", tkSlice)
	}
}

// RspUploadSlicesWrong updates the destination of slices for an ongoing upload
func RspUploadSlicesWrong(ctx context.Context, _ core.WriteCloser) {
	var target protos.RspUploadSlicesWrong
	if err := VerifyMessage(ctx, header.RspUploadSlicesWrong, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	rspUploadFile := target.RspUploadFile
	value, ok := task.UploadFileTaskMap.Load(rspUploadFile.FileHash)
	if !ok {
		pp.ErrorLogf(ctx, "File upload task cannot be found for file %v", rspUploadFile.FileHash)
		return
	}
	uploadTask := value.(*task.UploadFileTask)

	if target.Result.State != protos.ResultState_RES_SUCCESS {
		pp.ErrorLog(ctx, "RspUploadSlicesWrong failure:", target.Result.Msg)
		uploadTask.SetFatalError(errors.New(target.Result.Msg))
		return
	}

	if len(rspUploadFile.Slices) == 0 {
		pp.ErrorLogf(ctx, "No new slices in RspUploadSlicesWrong for file %v. Cannot update slice destinations", rspUploadFile.FileHash)
		uploadTask.SetFatalError(errors.New("No new slices in RspUploadSlicesWrong for file"))
		return
	}

	uploadTask.UpdateSliceDestinationsForRetry(rspUploadFile.Slices)

	uploadTask.SetRspUploadFile(target.RspUploadFile)
	// Start upload for all new destinations
	uploadTask.SignalNewDestinations(ctx)
}

// RspReportUploadSliceResult  SP-P OR SP-PP
func RspReportUploadSliceResult(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get RspReportUploadSliceResult")
	var target protos.RspReportUploadSliceResult
	if err := VerifyMessage(ctx, header.RspReportUploadSliceResult, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	if target.Result.State == protos.ResultState_RES_SUCCESS {
		utils.DebugLog("ResultState_RES_SUCCESS, sliceNumber，storageAddress，walletAddress",
			target.Slice.SliceNumber, target.Slice.PpInfo.NetworkAddress, target.Slice.PpInfo.P2PAddress)
	} else {
		utils.Log("ResultState_RES_FAIL : ", target.Result.Msg)
	}
}

func uploadSlice(ctx context.Context, slice *protos.SliceHashAddr, tk *task.UploadSliceTask, fileHash, taskId string) error {
	tkDataLen := int(slice.SliceOffset.SliceOffsetEnd - slice.SliceOffset.SliceOffsetStart)
	storageP2pAddress := slice.PpInfo.P2PAddress
	storageNetworkAddress := slice.PpInfo.NetworkAddress
	sliceNumber := tk.SliceNumber

	utils.DebugLog("reqID-"+taskId+" =========", strconv.FormatInt(core.GetReqIdFromContext(ctx), 10))
	tkSliceUID := taskId + strconv.FormatUint(tk.SliceNumber, 10)
	tkSlice := TaskSlice{
		TkSliceUID: tkSliceUID,
		SliceType:  SliceUpload,
		fileHash:   fileHash,
	}
	var ctStat = CostTimeStat{}

	_, data, err := file.ReadSliceDataFromTmp(fileHash, tk.SliceHash)
	if err != nil {
		return errors.Wrap(err, "failed to get slice data from tmp")
	}

	if tkDataLen <= setting.MaxData {
		packetId, newCtx := p2pserver.CreateNewContextPacketId(ctx)
		PacketIdMap.Store(packetId, tkSlice)
		utils.DebugLogf("PacketIdMap.Store <==(%v, %v)", packetId, tkSlice)
		ctStat.PacketCount = ctStat.PacketCount + 1
		UpSendCostTimeMap.mux.Lock()
		UpSendCostTimeMap.dataMap.Store(tkSliceUID, ctStat)
		UpSendCostTimeMap.mux.Unlock()
		pieceOffset := &protos.SliceOffset{
			SliceOffsetStart: uint64(0),
			SliceOffsetEnd:   uint64(tkDataLen),
		}
		var pb proto.Message
		var cmd header.MsgType
		utils.DebugLogf("upSendPacketWgMap.Store <== K:%v, V:%v]", tkSliceUID, ctStat)
		if tk.Type == protos.UploadType_BACKUP {
			pb = requests.ReqBackupFileSliceData(ctx, tk, pieceOffset, data[0])
			cmd = header.ReqBackupFileSlice
		} else {
			pb = requests.ReqUploadFileSliceData(ctx, tk, pieceOffset, data[0])
			cmd = header.ReqUploadFileSlice
		}
		return sendSlice(newCtx, pb, fileHash, storageP2pAddress, storageNetworkAddress, cmd)
	}

	// initialize cost time map, or clean the entry if this is a retry for the same slice
	UpSendCostTimeMap.mux.Lock()
	UpSendCostTimeMap.dataMap.Store(tkSliceUID, ctStat)
	UpSendCostTimeMap.mux.Unlock()

	dataStart := 0
	dataEnd := setting.MaxData
	for _, packet := range data {
		pieceOffset := &protos.SliceOffset{
			SliceOffsetStart: uint64(dataStart),
			SliceOffsetEnd:   uint64(dataEnd),
		}
		packetId, newCtx := p2pserver.CreateNewContextPacketId(ctx)
		PacketIdMap.Store(packetId, tkSlice)
		utils.DebugLogf("PacketIdMap.Store <==(%v, %v)", packetId, tkSlice)
		UpSendCostTimeMap.mux.Lock()

		if val, ok := UpSendCostTimeMap.dataMap.Load(taskId + strconv.FormatUint(sliceNumber, 10)); ok {
			ctStat = val.(CostTimeStat)
		}
		ctStat.PacketCount = ctStat.PacketCount + 1
		UpSendCostTimeMap.dataMap.Store(tkSliceUID, ctStat)
		UpSendCostTimeMap.mux.Unlock()
		utils.DebugLogf("upSendPacketMap.Store <== K:%v, V:%v]", tkSliceUID, ctStat)
		var cmd header.MsgType
		if dataEnd <= tkDataLen {
			utils.DebugLogf("Uploading slice data %v-%v (total %v)", dataStart, dataEnd, tkDataLen)
			var pb proto.Message
			if tk.Type == protos.UploadType_BACKUP {
				pb = requests.ReqBackupFileSliceData(ctx, tk, pieceOffset, packet)
				cmd = header.ReqBackupFileSlice
			} else {
				pb = requests.ReqUploadFileSliceData(ctx, tk, pieceOffset, packet)
				cmd = header.ReqUploadFileSlice
			}
			err := sendSlice(newCtx, pb, fileHash, storageP2pAddress, storageNetworkAddress, cmd)
			if err != nil {
				return err
			}
			dataStart += setting.MaxData
			dataEnd += setting.MaxData
		} else {
			utils.DebugLogf("Uploading slice data %v-%v (total %v)", dataStart, tkDataLen, tkDataLen)
			var pb proto.Message
			pieceOffset.SliceOffsetEnd = uint64(tkDataLen)
			if tk.Type == protos.UploadType_BACKUP {
				pb = requests.ReqBackupFileSliceData(ctx, tk, pieceOffset, packet)
				cmd = header.ReqBackupFileSlice
			} else {
				pb = requests.ReqUploadFileSliceData(ctx, tk, pieceOffset, packet)
				cmd = header.ReqUploadFileSlice
			}
			return sendSlice(newCtx, pb, fileHash, storageP2pAddress, storageNetworkAddress, cmd)
		}
	}
	return nil
}

func BackupFileSlice(ctx context.Context, tk *task.UploadSliceTask) error {
	var slice *protos.SliceHashAddr
	for _, slice = range tk.RspBackupFile.Slices {
		if slice.SliceNumber == tk.SliceNumber {
			break
		}
	}
	fileHash := tk.RspBackupFile.FileHash
	taskId := tk.RspBackupFile.TaskId
	err := uploadSlice(ctx, slice, tk, fileHash, taskId)
	return err
}

func UploadFileSlice(ctx context.Context, tk *task.UploadSliceTask) error {
	var slice *protos.SliceHashAddr
	for _, slice = range tk.RspUploadFile.Slices {
		if slice.SliceNumber == tk.SliceNumber {
			break
		}
	}
	fileHash := tk.RspUploadFile.FileHash
	taskId := tk.RspUploadFile.TaskId
	err := uploadSlice(ctx, slice, tk, fileHash, taskId)
	return err
}

func sendSlice(ctx context.Context, pb proto.Message, fileHash, p2pAddress, networkAddress string, cmd header.MsgType) error {
	utils.DebugLog("sendSlice(pb proto.MsgType, fileHash, p2pAddress, networkAddress string)", fileHash, p2pAddress, networkAddress)
	key := "upload#" + fileHash + p2pAddress
	//msg := pb.(*protos.ReqUploadFileSlice)
	//metrics.UploadPerformanceLogNow(fileHash + ":SND_FILE_DATA:" + strconv.FormatInt(int64(msg.PieceOffset.SliceOffsetStart+(msg.SliceNumber-1)*33554432), 10) + ":" + networkAddress)
	return p2pserver.GetP2pServer(ctx).SendMessageByCachedConn(ctx, key, networkAddress, pb, cmd, HandleSendPacketCostTime)
}

func UploadSpeedOfProgress(ctx context.Context, _ core.WriteCloser) {
	var target protos.UploadSpeedOfProgress
	if err := VerifyMessage(ctx, header.UploadSpeedOfProgress, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	prg, ok := task.UploadProgressMap.Load(target.FileHash)
	if !ok {
		utils.DebugLog(ctx, "can't load upload progress map...")
		return
	}
	metrics.UploadPerformanceLogNow(target.FileHash + ":RCV_PROGRESS:" + strconv.FormatInt(int64(target.SliceOffStart), 10))
	metrics.UploadPerformanceLogData(target.FileHash+":RCV_PROGRESS_DETAIL:"+strconv.FormatInt(int64(target.SliceOffStart), 10), target.HandleTime)
	progress := prg.(*task.UploadProgress)
	progress.HasUpload += int64(target.SliceSize)
	//p := float32(progress.HasUpload) / float32(progress.Total) * 100
	//pp.Logf(ctx, "fileHash: %v  uploaded：%.2f %% ", target.FileHash, p)
	//setting.ShowProgress(ctx, p)
	//ProgressMap.Store(target.FileHash, p)
	if progress.HasUpload >= progress.Total {
		task.UploadProgressMap.Delete(target.FileHash)
		p2pserver.GetP2pServer(ctx).CleanUpConnMap(target.FileHash)
		ScheduleReqBackupStatus(ctx, target.FileHash)
	}
}

func HandleSendPacketCostTime(ctx context.Context, packetId, costTime int64, conn core.WriteCloser) {
	if packetId <= 0 || costTime <= 0 {
		return
	}
	mutexHandleSendCostTime.Lock()
	defer mutexHandleSendCostTime.Unlock()
	// get record by reqId
	if val, ok := PacketIdMap.Load(packetId); ok {
		utils.DebugLogf("get packetId[%v] from PacketIdMap, success", packetId)
		tkSlice := val.(TaskSlice)
		if len(tkSlice.TkSliceUID) > 0 {
			PacketIdMap.Delete(packetId)
		}
		var sliceType string
		switch tkSlice.SliceType {
		case SliceInvalid:
		case SliceUpload:
			sliceType = "Upload"
			go handleUploadSend(tkSlice, costTime)
		case SliceDownload:
			sliceType = "Download"
			// a storage PP is on the write side. The failure is reported by the downloader.
			go handleDownloadSend(tkSlice, costTime)
		case SliceBackup, SliceTransfer:
			sliceType = "Transfer/Backup"
			// tell sp about the failure
			if costTime > (time.Duration(utils.WriteTimeOut) * time.Second).Milliseconds() {
				SendReportBackupSliceResult(ctx, tkSlice.TaskId, tkSlice.SliceHash, tkSlice.SpP2pAddress, false, tkSlice.OriginDeleted, costTime)
			}
			go handleBackupTransferSend(tkSlice, costTime)
		}

		if costTime > (time.Duration(utils.WriteTimeOut) * time.Second).Milliseconds() {
			utils.DebugLog("Closing a slow connection during", sliceType)
			go conn.Close()
		}
	}
}

func handleUploadSend(tkSlice TaskSlice, costTime int64) {
	newCostTimeStat := CostTimeStat{}
	UpSendCostTimeMap.mux.Lock()
	defer UpSendCostTimeMap.mux.Unlock()
	if val, ok := UpSendCostTimeMap.dataMap.Load(tkSlice.TkSliceUID); ok {
		utils.DebugLogf("get TkSliceUID[%v] from dataMap, success", tkSlice.TkSliceUID)
		oriCostTimeStat := val.(CostTimeStat)
		newCostTimeStat.TotalCostTime = costTime + oriCostTimeStat.TotalCostTime
		newCostTimeStat.PacketCount = oriCostTimeStat.PacketCount - 1
		// not counting if CostTimeStat not found from dataMap
		if newCostTimeStat.PacketCount >= 0 {
			UpSendCostTimeMap.dataMap.Store(tkSlice.TkSliceUID, newCostTimeStat)
			utils.DebugLogf("newCostTimeStat is %v", newCostTimeStat)
		}
	} else {
		utils.DebugLogf("get TkSliceUID[%v] from dataMap, fail", tkSlice.TkSliceUID)
	}
}
