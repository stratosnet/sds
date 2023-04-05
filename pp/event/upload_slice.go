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
	"github.com/stratosnet/sds/utils/types"
	"google.golang.org/protobuf/proto"
)

var (
	//// ProgressMap required by API
	//ProgressMap             = &sync.Map{}
	mutexHandleSendCostTime = &sync.Mutex{}
	//// Maps to record uploading stats
	PacketIdMap       = &sync.Map{} // K: reqId, V: TaskSlice{tkId+sliceNum, up/down}
	upSendCostTimeMap = &upSendCostTime{
		dataMap: utils.NewAutoCleanUnsafeMap(30 * time.Minute), // make(map[string]*CostTimeStat) // K: tkId+sliceNum, V: CostTimeStat{TotalCostTime, PacketCount}
		mux:     sync.Mutex{},
	}
	upRecvCostTimeMap = &upRecvCostTime{
		dataMap: utils.NewAutoCleanUnsafeMap(30 * time.Minute), // make(map[string]int64), // K: tkId+sliceNum, V: TotalCostTime
		mux:     sync.Mutex{},
	}
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
	TkSliceUID string
	IsUpload   bool
}

type CostTimeStat struct {
	TotalCostTime int64
	PacketCount   int64
}

func GetOngoingUploadTaskCount() int {
	upRecvCostTimeMap.mux.Lock()
	count := upRecvCostTimeMap.dataMap.Len()
	upRecvCostTimeMap.mux.Unlock()
	return count
}

// ReqUploadFileSlice storage PP receives a request with file data from the PP who initiated uploading
func ReqUploadFileSlice(ctx context.Context, conn core.WriteCloser) {
	costTime := core.GetRecvCostTimeFromContext(ctx)
	var target protos.ReqUploadFileSlice
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	// spam check after verified the sp's response
	if time.Now().Unix()-target.RspUploadFile.TimeStamp > setting.SPAM_THRESHOLD_SP_SIGN_LATENCY {
		rsp := &protos.RspUploadFileSlice{
			Result: &protos.Result{
				State: protos.ResultState_RES_FAIL,
				Msg:   "sp's upload file response was expired",
			},
		}
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspUploadFileSlice)
		return
	}

	// check if signatures exist
	if target.P2PAddress == "" || target.RspUploadFile.TimeStamp == 0 {
		rsp := &protos.RspUploadFileSlice{
			Result: &protos.Result{
				State: protos.ResultState_RES_FAIL,
				Msg:   "missing information for verification",
			},
		}
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspUploadFileSlice)
		return
	}

	// verify addresses and signatures
	if err := verifyUploadSliceSign(&target); err != nil {
		rsp := &protos.RspUploadFileSlice{
			Result: &protos.Result{
				State: protos.ResultState_RES_FAIL,
				Msg:   err.Error(),
			},
		}
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspUploadFileSlice)
		return
	}

	// setting local variables
	fileHash := target.RspUploadFile.FileHash
	var slice *protos.SliceHashAddr
	for _, slice = range target.RspUploadFile.Slices {
		if slice.SliceNumber == target.SliceNumber {
			break
		}
	}
	newSlice := slice
	sliceSizeFromMsg := slice.SliceOffset.SliceOffsetEnd - slice.SliceOffset.SliceOffsetStart

	if slice.PpInfo.P2PAddress != setting.P2PAddress {
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
	tkSlice := target.RspUploadFile.TaskId + strconv.FormatUint(target.SliceNumber, 10)
	upRecvCostTimeMap.mux.Lock()
	if val, ok := upRecvCostTimeMap.dataMap.Load(tkSlice); ok {
		totalCostTime += val.(int64)
	}
	upRecvCostTimeMap.dataMap.Store(tkSlice, totalCostTime)
	upRecvCostTimeMap.mux.Unlock()
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
			_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, requests.RspUploadFileSliceData(&target), header.RspUploadFileSlice)
			// report upload result to SP
			newSlice.SliceHash = target.SliceHash
			_, newCtx := p2pserver.CreateNewContextPacketId(ctx)
			utils.DebugLog("ReqReportUploadSliceResultDataPP reqID =========", core.GetReqIdFromContext(newCtx))
			reportResultReq := requests.ReqReportUploadSliceResultData(target.RspUploadFile.TaskId,
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
			upRecvCostTimeMap.mux.Lock()
			upRecvCostTimeMap.dataMap.Delete(tkSlice)
			upRecvCostTimeMap.mux.Unlock()
			utils.DebugLog("storage PP report to SP upload task finished: ", target.SliceHash)
		} else {
			utils.ErrorLog("newly stored sliceHash is not equal to target sliceHash!")
		}
	}
}

func RspUploadFileSlice(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspUploadFileSlice
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
	upSendCostTimeMap.mux.Lock()
	defer upSendCostTimeMap.mux.Unlock()
	if val, ok := upSendCostTimeMap.dataMap.Load(tkSlice); ok {
		ctStat := val.(CostTimeStat)
		utils.DebugLogf("ctStat is %v", ctStat)
		if ctStat.PacketCount == 0 && ctStat.TotalCostTime > 0 {
			target.Slice.SliceHash = target.SliceHash
			reportReq := requests.ReqReportUploadSliceResultData(
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

			upSendCostTimeMap.dataMap.Delete(tkSlice)
		}
	} else {
		utils.DebugLogf("tkSlice [%v] not found in RspUploadFileSlice", tkSlice)
	}
}

// ReqBackupFileSlice
func ReqBackupFileSlice(ctx context.Context, conn core.WriteCloser) {
	costTime := core.GetRecvCostTimeFromContext(ctx)
	var target protos.ReqBackupFileSlice
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	// spam check after verified the sp's response
	if time.Now().Unix()-target.RspBackupFile.TimeStamp > setting.SPAM_THRESHOLD_SP_SIGN_LATENCY {
		rsp := &protos.RspUploadFileSlice{
			Result: &protos.Result{
				State: protos.ResultState_RES_FAIL,
				Msg:   "sp's upload file response was expired",
			},
		}
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspBackupFileSlice)
		return
	}

	// verify addresses and signatures
	if err := verifyBackupSliceSign(&target); err != nil {
		rsp := &protos.RspUploadFileSlice{
			Result: &protos.Result{
				State: protos.ResultState_RES_FAIL,
				Msg:   err.Error(),
			},
		}
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspBackupFileSlice)
		return
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

	if slice.PpInfo.P2PAddress != setting.P2PAddress {
		rsp := &protos.RspUploadFileSlice{
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
	upRecvCostTimeMap.mux.Lock()
	if val, ok := upRecvCostTimeMap.dataMap.Load(tkSlice); ok {
		totalCostTime += val.(int64)
	}
	upRecvCostTimeMap.dataMap.Store(tkSlice, totalCostTime)
	upRecvCostTimeMap.mux.Unlock()
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
			reportResultReq := requests.ReqReportUploadSliceResultData(target.RspBackupFile.TaskId,
				target.RspBackupFile.FileHash,
				target.RspBackupFile.SpP2PAddress,
				setting.P2PAddress,
				true,
				slice,
				totalCostTime)
			p2pserver.GetP2pServer(ctx).SendMessageToSPServer(newCtx, reportResultReq, header.ReqReportUploadSliceResult)
			metrics.StoredSliceCount.WithLabelValues("upload").Inc()
			instantInboundSpeed := float64(sliceSizeFromMsg) / math.Max(float64(totalCostTime), 1)
			metrics.InboundSpeed.WithLabelValues(reportResultReq.OpponentP2PAddress).Set(instantInboundSpeed)
			upRecvCostTimeMap.mux.Lock()
			upRecvCostTimeMap.dataMap.Delete(tkSlice)
			upRecvCostTimeMap.mux.Unlock()
			utils.DebugLog("storage PP report to SP upload task finished: ", target.SliceHash)
		} else {
			utils.ErrorLog("newly stored sliceHash is not equal to target sliceHash!")
		}
	}
}

func RspBackupFileSlice(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspBackupFileSlice
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	pp.DebugLogf(ctx, "get RspUploadFileSlice for file %v  sliceNumber %v  size %v", target.FileHash, target.Slice.SliceNumber, target.SliceSize)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		pp.ErrorLog(ctx, "RspUploadFileSlice failure:", target.Result.Msg)
		return
	}
	tkSlice := target.TaskId + strconv.FormatUint(target.Slice.SliceNumber, 10)
	upSendCostTimeMap.mux.Lock()
	defer upSendCostTimeMap.mux.Unlock()
	if val, ok := upSendCostTimeMap.dataMap.Load(tkSlice); ok {
		ctStat := val.(CostTimeStat)
		utils.DebugLogf("ctStat is %v", ctStat)
		if ctStat.PacketCount == 0 && ctStat.TotalCostTime > 0 {
			reportReq := requests.ReqReportUploadSliceResultData(target.TaskId,
				target.FileHash,
				target.SpP2PAddress,
				setting.P2PAddress,
				false,
				target.Slice,
				ctStat.TotalCostTime)

			p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, reportReq, header.ReqReportUploadSliceResult)
			instantOutboundSpeed := float64(target.SliceSize) / math.Max(float64(ctStat.TotalCostTime), 1)
			metrics.OutboundSpeed.WithLabelValues(target.P2PAddress).Set(instantOutboundSpeed)

			upSendCostTimeMap.dataMap.Delete(tkSlice)
		}
	} else {
		utils.DebugLogf("tkSlice [%v] not found in RspUploadFileSlice", tkSlice)
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
		TkSliceUID: tkSliceUID,
		IsUpload:   true,
	}
	var ctStat = CostTimeStat{}
	if tkDataLen <= setting.MAXDATA {
		data, err := file.GetSliceDataFromTmp(fileHash, tk.SliceHash)
		if err != nil {
			return errors.Wrap(err, "failed get slice data from tmp")
		}
		packetId, newCtx := p2pserver.CreateNewContextPacketId(ctx)
		PacketIdMap.Store(packetId, tkSlice)
		utils.DebugLogf("PacketIdMap.Store <==(%v, %v)", packetId, tkSlice)
		ctStat.PacketCount = ctStat.PacketCount + 1
		upSendCostTimeMap.mux.Lock()
		upSendCostTimeMap.dataMap.Store(tkSliceUID, ctStat)
		upSendCostTimeMap.mux.Unlock()
		pieceOffset := &protos.SliceOffset{
			SliceOffsetStart: uint64(0),
			SliceOffsetEnd:   uint64(tkDataLen),
		}
		var pb proto.Message
		var cmd string
		utils.DebugLogf("upSendPacketWgMap.Store <== K:%v, V:%v]", tkSliceUID, ctStat)
		if tk.Type == protos.UploadType_BACKUP {
			pb = requests.ReqBackupFileSliceData(tk, storageP2pAddress, pieceOffset, data)
			cmd = header.ReqBackupFileSlice
		} else {
			pb = requests.ReqUploadFileSliceData(tk, storageP2pAddress, pieceOffset, data)
			cmd = header.ReqUploadFileSlice
		}
		return sendSlice(newCtx, pb, fileHash, storageP2pAddress, cmd, storageNetworkAddress)
	}

	data, err := file.GetSliceDataFromTmp(fileHash, tk.SliceHash)
	if err != nil {
		return errors.Wrap(err, "failed get slice data from tmp")
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
		upSendCostTimeMap.mux.Lock()

		if val, ok := upSendCostTimeMap.dataMap.Load(taskId + strconv.FormatUint(sliceNumber, 10)); ok {
			ctStat = val.(CostTimeStat)
		}
		ctStat.PacketCount = ctStat.PacketCount + 1
		upSendCostTimeMap.dataMap.Store(tkSliceUID, ctStat)
		upSendCostTimeMap.mux.Unlock()
		utils.DebugLogf("upSendPacketMap.Store <== K:%v, V:%v]", tkSliceUID, ctStat)
		var cmd string
		if dataEnd < (tkDataLen + 1) {
			pp.DebugLogf(newCtx, "Uploading slice data %v-%v (total %v)", dataStart, dataEnd, tkDataLen)
			var pb proto.Message
			if tk.Type == protos.UploadType_BACKUP {
				pb = requests.ReqBackupFileSliceData(tk, storageP2pAddress, pieceOffset, data[dataStart:dataEnd])
				cmd = header.ReqBackupFileSlice
			} else {
				pb = requests.ReqUploadFileSliceData(tk, storageP2pAddress, pieceOffset, data[dataStart:dataEnd])
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
				pb = requests.ReqBackupFileSliceData(tk, storageP2pAddress, pieceOffset, data[dataStart:])
				cmd = header.ReqBackupFileSlice
			} else {
				pb = requests.ReqUploadFileSliceData(tk, storageP2pAddress, pieceOffset, data[dataStart:])
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
		utils.DebugLogf("slice.SliceNumber:", slice.SliceNumber)
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
		utils.DebugLogf("slice.SliceNumber:", slice.SliceNumber)
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

func verifyUploadSliceSign(target *protos.ReqUploadFileSlice) error {
	rspUploadFile := target.RspUploadFile

	spP2pPubkey, err := requests.GetSpPubkey(target.RspUploadFile.SpP2PAddress)
	if err != nil {
		return errors.Wrap(err, "failed to get sp pubkey")
	}

	// verify sp address
	if !types.VerifyP2pAddrBytes(spP2pPubkey, target.RspUploadFile.SpP2PAddress) {
		return errors.New("failed verifying sp's p2p address")
	}

	// verify sp node signature
	nodeSign := rspUploadFile.NodeSign
	rspUploadFile.NodeSign = nil
	signmsg, err := utils.GetRspUploadFileSpNodeSignMessage(rspUploadFile)
	if err != nil {
		return errors.New("failed getting sp's sign message")
	}
	if !types.VerifyP2pSignBytes(spP2pPubkey, nodeSign, signmsg) {
		return errors.New("failed verifying sp's signature")
	}
	rspUploadFile.NodeSign = nodeSign
	return nil
}

func verifyBackupSliceSign(target *protos.ReqBackupFileSlice) error {
	rspBackupFile := target.RspBackupFile

	spP2pPubkey, err := requests.GetSpPubkey(target.RspBackupFile.SpP2PAddress)
	if err != nil {
		return errors.Wrap(err, "failed to get sp pubkey")
	}

	// verify sp address
	if !types.VerifyP2pAddrBytes(spP2pPubkey, target.RspBackupFile.SpP2PAddress) {
		return errors.New("failed verifying sp's p2p address")
	}
	time.Unix(rspBackupFile.TimeStamp, 0).String()
	// verify sp node signature
	nodeSign := rspBackupFile.NodeSign
	rspBackupFile.NodeSign = nil
	signmsg, err := utils.GetRspBackupFileSpNodeSignMessage(rspBackupFile)
	if err != nil {
		return errors.New("failed getting sp's sign message")
	}
	if !types.VerifyP2pSignBytes(spP2pPubkey, nodeSign, signmsg) {
		return errors.New("failed verifying sp's signature")
	}
	rspBackupFile.NodeSign = nodeSign
	return nil
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
		utils.DebugLogf("HandleSendPacketCostTime, packetId=%v, isUpload=%v, newReport.costTime=%v, ", packetId, tkSlice.IsUpload, costTime)
		if tkSlice.IsUpload {
			go handleUploadSend(tkSlice, costTime)
		} else {
			go handleDownloadSend(tkSlice, costTime)
		}
	}
}

func handleUploadSend(tkSlice TaskSlice, costTime int64) {
	var newCostTimeStat = CostTimeStat{}
	upSendCostTimeMap.mux.Lock()
	defer upSendCostTimeMap.mux.Unlock()
	if val, ok := upSendCostTimeMap.dataMap.Load(tkSlice.TkSliceUID); ok {
		utils.DebugLogf("get TkSliceUID[%v] from dataMap, success", tkSlice.TkSliceUID)
		oriCostTimeStat := val.(CostTimeStat)
		newCostTimeStat.TotalCostTime = costTime + oriCostTimeStat.TotalCostTime
		newCostTimeStat.PacketCount = oriCostTimeStat.PacketCount - 1
		// not counting if CostTimeStat not found from dataMap
		if newCostTimeStat.PacketCount >= 0 {
			upSendCostTimeMap.dataMap.Store(tkSlice.TkSliceUID, newCostTimeStat)
			utils.DebugLogf("newCostTimeStat is %v", newCostTimeStat)
		}
	} else {
		utils.DebugLogf("get TkSliceUID[%v] from dataMap, fail", tkSlice.TkSliceUID)
	}
}
