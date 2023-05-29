package event

// Author j
import (
	"context"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/metrics"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils"
	"google.golang.org/protobuf/proto"
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

	uploadSliceSpamCheckMap = utils.NewAutoCleanMap(setting.SPAM_THRESHOLD_SLICE_OPERATIONS)
	backupSliceSpamCheckMap = utils.NewAutoCleanMap(setting.SPAM_THRESHOLD_SLICE_OPERATIONS)
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
	TkSliceUID         string
	IsUpload           bool
	IsBackupOrTransfer bool
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

	rspUploadFile := target.RspUploadFile
	var slice *protos.SliceHashAddr
	for _, slice = range rspUploadFile.Slices {
		if slice.SliceNumber == target.SliceNumber {
			break
		}
	}

	if target.PieceOffset.SliceOffsetStart >= slice.SliceSize || target.PieceOffset.SliceOffsetStart%setting.MAXDATA != 0 {
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
	key := rspUploadFile.TaskId + strconv.FormatInt(int64(target.SliceNumber), 10) + strconv.FormatInt(int64(target.PieceOffset.SliceOffsetStart), 10)
	if _, ok := uploadSliceSpamCheckMap.Load(key); ok {
		rsp := &protos.RspUploadFileSlice{
			Result: &protos.Result{
				State: protos.ResultState_RES_FAIL,
				Msg:   "failed uploading file slice, re-upload",
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

	if slice.PpInfo.P2PAddress != p2pserver.GetP2pServer(ctx).GetP2PAddress() {
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
	_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, requests.UploadSpeedOfProgressData(fileHash, uint64(len(target.Data)), (target.SliceNumber-1)*33554432+target.PieceOffset.SliceOffsetStart, timeEntry), header.UploadSpeedOfProgress)
	if err := task.SaveUploadFile(&target); err != nil {
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
		if utils.CalcSliceHash(sliceData, fileHash, target.SliceNumber) == target.SliceHash {
			_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, requests.RspUploadFileSliceData(ctx, &target), header.RspUploadFileSlice)
			// report upload result to SP
			newSlice.SliceHash = target.SliceHash
			_, newCtx := p2pserver.CreateNewContextPacketId(ctx)
			utils.DebugLog("ReqReportUploadSliceResultDataPP reqID =========", core.GetReqIdFromContext(newCtx))
			reportResultReq := requests.ReqReportUploadSliceResultData(ctx, target.RspUploadFile.TaskId,
				target.RspUploadFile.FileHash,
				target.RspUploadFile.SpP2PAddress,
				target.P2PAddress,
				true,
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
		}
	}
}

func RspUploadFileSlice(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspUploadFileSlice
	if err := VerifyMessage(ctx, header.RspUploadFileSlice, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	metrics.UploadPerformanceLogNow(target.FileHash + ":RCV_RSP_SLICE:" + strconv.FormatInt(int64(target.Slice.SliceNumber), 10))

	pp.DebugLogf(ctx, "get RspUploadFileSlice for file %v  sliceNumber %v  size %v",
		target.FileHash, target.Slice.SliceNumber, target.Slice.SliceSize)
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
			target.Slice.SliceHash = target.SliceHash
			reportReq := requests.ReqReportUploadSliceResultData(ctx,
				target.TaskId,
				target.FileHash,
				target.SpP2PAddress,
				target.Slice.PpInfo.P2PAddress,
				false,
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

	// spam check
	key := target.RspBackupFile.TaskId + strconv.FormatInt(int64(target.SliceNumber), 10)
	if _, ok := backupSliceSpamCheckMap.Load(key); ok {
		rsp := &protos.RspBackupFileSlice{
			Result: &protos.Result{
				State: protos.ResultState_RES_FAIL,
				Msg:   "failed backing up file slice, re-backup",
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

	if slice.PpInfo.P2PAddress != p2pserver.GetP2pServer(ctx).GetP2PAddress() {
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
		(target.SliceNumber-1)*setting.MAX_SLICE_SIZE+target.PieceOffset.SliceOffsetStart, timeEntry)
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
		if utils.CalcSliceHash(sliceData, fileHash, target.SliceNumber) == target.SliceHash {
			_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, requests.RspBackupFileSliceData(&target), header.RspBackupFileSlice)
			// report upload result to SP

			_, newCtx := p2pserver.CreateNewContextPacketId(ctx)
			utils.DebugLog("ReqReportUploadSliceResultDataPP reqID =========", core.GetReqIdFromContext(newCtx))
			reportResultReq := requests.ReqReportUploadSliceResultData(
				ctx,
				target.RspBackupFile.TaskId,
				target.RspBackupFile.FileHash,
				target.RspBackupFile.SpP2PAddress,
				p2pserver.GetP2pServer(ctx).GetP2PAddress(),
				true,
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
				p2pserver.GetP2pServer(ctx).GetP2PAddress(),
				false,
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
		pp.ErrorLogf(ctx, "No new slices in RspUploadSlicesWrong for file %v. Cannot update slice destinations", target.FileHash)
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
	if err := VerifyMessage(ctx, header.RspReportUploadSliceResult, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	if target.Result.State == protos.ResultState_RES_SUCCESS {
		pp.DebugLog(ctx, "ResultState_RES_SUCCESS, sliceNumber，storageAddress，walletAddress",
			target.Slice.SliceNumber, target.Slice.PpInfo.NetworkAddress, target.Slice.PpInfo.P2PAddress)
	} else {
		pp.Log(ctx, "ResultState_RES_FAIL : ", target.Result.Msg)
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
		TkSliceUID:         tkSliceUID,
		IsUpload:           true,
		IsBackupOrTransfer: false,
	}
	var ctStat = CostTimeStat{}

	data, err := file.GetSliceDataFromTmp(fileHash, tk.SliceHash)
	if err != nil {
		return errors.Wrap(err, "failed to get slice data from tmp")
	}

	if tkDataLen <= setting.MAXDATA {
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
		var cmd string
		utils.DebugLogf("upSendPacketWgMap.Store <== K:%v, V:%v]", tkSliceUID, ctStat)
		p2pAddress := p2pserver.GetP2pServer(ctx).GetP2PAddress()
		if tk.Type == protos.UploadType_BACKUP {
			pb = requests.ReqBackupFileSliceData(ctx, tk, pieceOffset, data)
			cmd = header.ReqBackupFileSlice
		} else {
			pb = requests.ReqUploadFileSliceData(ctx, tk, pieceOffset, data)
			cmd = header.ReqUploadFileSlice
		}
		return sendSlice(newCtx, pb, fileHash, p2pAddress, cmd, storageNetworkAddress)
	}

	dataStart := 0
	dataEnd := setting.MAXDATA
	for {
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
		var cmd string
		if dataEnd < (tkDataLen + 1) {
			pp.DebugLogf(newCtx, "Uploading slice data %v-%v (total %v)", dataStart, dataEnd, tkDataLen)
			var pb proto.Message
			if tk.Type == protos.UploadType_BACKUP {
				pb = requests.ReqBackupFileSliceData(ctx, tk, pieceOffset, data[dataStart:dataEnd])
				cmd = header.ReqBackupFileSlice
			} else {
				pb = requests.ReqUploadFileSliceData(ctx, tk, pieceOffset, data[dataStart:dataEnd])
				cmd = header.ReqUploadFileSlice
			}
			err := sendSlice(newCtx, pb, fileHash, storageP2pAddress, cmd, storageNetworkAddress)
			if err != nil {
				return err
			}
			dataStart += setting.MAXDATA
			dataEnd += setting.MAXDATA
		} else {
			pp.DebugLogf(newCtx, "Uploading slice data %v-%v (total %v)", dataStart, tkDataLen, tkDataLen)
			var pb proto.Message
			if tk.Type == protos.UploadType_BACKUP {
				pb = requests.ReqBackupFileSliceData(ctx, tk, pieceOffset, data[dataStart:])
				cmd = header.ReqBackupFileSlice
			} else {
				pb = requests.ReqUploadFileSliceData(ctx, tk, pieceOffset, data[dataStart:])
				cmd = header.ReqUploadFileSlice
			}
			return sendSlice(newCtx, pb, fileHash, storageP2pAddress, cmd, storageNetworkAddress)
		}
	}
}

func BackupFileSlice(ctx context.Context, tk *task.UploadSliceTask) error {
	var slice *protos.SliceHashAddr
	utils.DebugLog("tk.SliceNumber:", tk.SliceNumber)
	utils.DebugLog(tk)
	for _, slice = range tk.RspBackupFile.Slices {
		utils.DebugLogf("slice.SliceNumber: %d", slice.SliceNumber)
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
	utils.DebugLog("tk.SliceNumber:", tk.SliceNumber)
	utils.DebugLog(tk)
	for _, slice = range tk.RspUploadFile.Slices {
		utils.DebugLogf("slice.SliceNumber: %d", slice.SliceNumber)
		if slice.SliceNumber == tk.SliceNumber {
			break
		}
	}
	fileHash := tk.RspUploadFile.FileHash
	taskId := tk.RspUploadFile.TaskId
	err := uploadSlice(ctx, slice, tk, fileHash, taskId)
	return err
}

func sendSlice(ctx context.Context, pb proto.Message, fileHash, p2pAddress, cmd, networkAddress string) error {
	pp.DebugLog(ctx, "sendSlice(pb proto.Message, fileHash, p2pAddress, networkAddress string)",
		fileHash, p2pAddress, networkAddress)
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
		pp.DebugLog(ctx, "paused!!")
		return
	}
	metrics.UploadPerformanceLogNow(target.FileHash + ":RCV_PROGRESS:" + strconv.FormatInt(int64(target.SliceOffStart), 10))
	metrics.UploadPerformanceLogData(target.FileHash+":RCV_PROGRESS_DETAIL:"+strconv.FormatInt(int64(target.SliceOffStart), 10), target.HandleTime)
	progress := prg.(*task.UploadProgress)
	progress.HasUpload += int64(target.SliceSize)
	p := float32(progress.HasUpload) / float32(progress.Total) * 100
	pp.Logf(ctx, "fileHash: %v  uploaded：%.2f %% ", target.FileHash, p)
	setting.ShowProgress(ctx, p)
	//ProgressMap.Store(target.FileHash, p)
	if progress.HasUpload >= progress.Total {
		task.UploadProgressMap.Delete(target.FileHash)
		p2pserver.GetP2pServer(ctx).CleanUpConnMap(target.FileHash)
		ScheduleReqBackupStatus(ctx, target.FileHash)
		if file.IsFileRpcRemote(target.FileHash) {
			file.SetRemoteFileResult(target.FileHash, rpc.Result{Return: rpc.SUCCESS})
		}
	}
}

func HandleSendPacketCostTime(packetId, costTime int64) {
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
		utils.DebugLogf("HandleSendPacketCostTime, packetId=%v, isUpload=%v, isBackupOrTransfer=%v, newReport.costTime=%v, ",
			packetId, tkSlice.IsUpload, tkSlice.IsBackupOrTransfer, costTime)
		if tkSlice.IsBackupOrTransfer {
			go handleBackupTransferSend(tkSlice, costTime)
		} else if tkSlice.IsUpload {
			go handleUploadSend(tkSlice, costTime)
		} else {
			go handleDownloadSend(tkSlice, costTime)
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
