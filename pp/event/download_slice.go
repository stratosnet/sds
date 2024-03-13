package event

// Author j cc
import (
	"context"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/crypto"
	"github.com/stratosnet/sds/framework/crypto/encryption"
	"github.com/stratosnet/sds/framework/crypto/encryption/hdkey"
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

const (
	LOSE_SLICE_MSG = "cannot find the file slice"
)

var (
	downloadRspMap      = &sync.Map{} // K: tkId+sliceHash, V: *QueuedDownloadReportToSP
	DownSendCostTimeMap = &downSendCostTime{
		dataMap: utils.NewAutoCleanUnsafeMap(30 * time.Minute), // make(map[string]*CostTimeStat), // K: tkId+sliceHash, V: CostTimeStat{TotalCostTime, PacketCount}
		mux:     sync.Mutex{},
	}
	DownRecvCostTimeMap = &downRecvCostTime{
		dataMap: utils.NewAutoCleanUnsafeMap(30 * time.Minute), // make(map[string]int64), // K: tkId+sliceHash, V: costTime in int64
		mux:     sync.Mutex{},
	}

	downloadSliceSpamCheckMap = utils.NewAutoCleanMap(setting.SpamThresholdSliceOperations)
)

type downSendCostTime struct {
	dataMap *utils.AutoCleanUnsafeMap // map[string]*CostTimeStat // K: tkId+sliceHash, V: CostTimeStat{TotalCostTime, PacketCount}
	mux     sync.Mutex
}
type downRecvCostTime struct {
	dataMap *utils.AutoCleanUnsafeMap // map[string]int64 // K: tkId+sliceHash, V: costTime in int64
	mux     sync.Mutex
}

func (dsc *downSendCostTime) StartSendPacket(tkSliceHashKey string) (costTimeStatAfter CostTimeStat) {
	dsc.mux.Lock()
	defer dsc.mux.Unlock()
	costTimeStatBefore := CostTimeStat{}
	costTimeStatAfter = CostTimeStat{}
	if val, ok := dsc.dataMap.Load(tkSliceHashKey); ok {
		costTimeStatBefore = val.(CostTimeStat)
	}
	costTimeStatAfter.TotalCostTime = costTimeStatBefore.TotalCostTime
	costTimeStatAfter.PacketCount = costTimeStatBefore.PacketCount + 1
	dsc.dataMap.Store(tkSliceHashKey, costTimeStatAfter)
	utils.DebugLogf("--- downSendCostTime.StartSendPacket --- got 1 new packet, packetCountAfter: %d ",
		costTimeStatAfter.PacketCount)
	return costTimeStatAfter
}

func (dsc *downSendCostTime) FinishSendPacket(tkSliceHashKey string, costTime int64) (costTimeStatAfter CostTimeStat) {
	dsc.mux.Lock()
	defer dsc.mux.Unlock()
	costTimeStatAfter = CostTimeStat{}
	if val, ok := dsc.dataMap.Load(tkSliceHashKey); ok {
		costTimeStatBefore := val.(CostTimeStat)
		costTimeStatAfter.TotalCostTime = costTimeStatBefore.TotalCostTime + costTime
		costTimeStatAfter.PacketCount = costTimeStatBefore.PacketCount - 1
		if costTimeStatAfter.PacketCount >= 0 {
			dsc.dataMap.Store(tkSliceHashKey, costTimeStatAfter)
		}
	}
	utils.DebugLogf("--- downSendCostTime.FinishSendPacket --- 1 new finished packet at costTime = %d ms, statAfter: %v",
		costTime, costTimeStatAfter)
	return costTimeStatAfter
}

func (dsc *downSendCostTime) CheckSendSliceCompletion(tkSliceHashKey string) bool {
	dsc.mux.Lock()
	defer dsc.mux.Unlock()
	if val, ok := dsc.dataMap.Load(tkSliceHashKey); ok {
		costTimeStat := val.(CostTimeStat)
		if costTimeStat.PacketCount == 0 && costTimeStat.TotalCostTime > 0 {
			return true
		}
	}
	return false
}

func (dsc *downSendCostTime) GetCompletedTotalCostTime(tkSliceHashKey string) (int64, bool) {
	dsc.mux.Lock()
	defer dsc.mux.Unlock()
	if val, ok := dsc.dataMap.Load(tkSliceHashKey); ok {
		costTimeStat := val.(CostTimeStat)
		if costTimeStat.PacketCount == 0 && costTimeStat.TotalCostTime > 0 {
			return costTimeStat.TotalCostTime, true
		}
	}
	return int64(0), false
}

func (dsc *downSendCostTime) DeleteRecord(tkSliceHashKey string) {
	dsc.mux.Lock()
	defer dsc.mux.Unlock()
	dsc.dataMap.Delete(tkSliceHashKey)
}

func (drc *downRecvCostTime) AddCostTime(tkSliceHashKey string, costTime int64) (totalCostTime int64) {
	drc.mux.Lock()
	defer drc.mux.Unlock()
	totalCostTime = costTime
	if val, ok := drc.dataMap.Load(tkSliceHashKey); ok {
		totalCostTime += val.(int64)
	}
	if costTime > 0 {
		drc.dataMap.Store(tkSliceHashKey, totalCostTime)
		utils.DebugLogf("--- downRecvCostTime.AddCostTime --- add %d ms from newly received packet, total: %d ms",
			costTime, totalCostTime)
	}
	return totalCostTime
}

func (drc *downRecvCostTime) DeleteRecord(tkSliceHashKey string) {
	drc.mux.Lock()
	defer drc.mux.Unlock()
	drc.dataMap.Delete(tkSliceHashKey)
}

type QueuedDownloadReportToSP struct {
	context  context.Context
	response *protos.RspDownloadSlice
}

func GetOngoingDownloadTaskCount() int {
	count := 0
	downloadRspMap.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}

func setWriteHookForRspDownloadSlice(conn core.WriteCloser) {
	switch conn := conn.(type) {
	case *core.ServerConn:
		hook := core.WriteHook{
			MessageId: header.RspDownloadSlice.Id,
			Fn:        HandleSendPacketCostTime,
		}
		var hooks []core.WriteHook
		hooks = append(hooks, hook)
		conn.SetWriteHook(hooks)
	case *cf.ClientConn:
		hook := cf.WriteHook{
			MessageId: header.RspDownloadSlice.Id,
			Fn:        HandleSendPacketCostTime,
		}
		var hooks []cf.WriteHook
		hooks = append(hooks, hook)
		conn.SetWriteHook(hooks)
	}
}

// ReqDownloadSlice download slice PP-storagePP
func ReqDownloadSlice(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("ReqDownloadSlice reqID =========", core.GetReqIdFromContext(ctx))
	utils.Log("ReqDownloadSlice", conn)
	var target protos.ReqDownloadSlice
	if err := VerifyMessage(ctx, header.ReqDownloadSlice, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	// spam check
	key := target.RspFileStorageInfo.TaskId + strconv.FormatInt(int64(target.SliceNumber), 10) +
		target.P2PAddress + strconv.FormatInt(target.RspFileStorageInfo.TimeStamp, 10)
	if _, ok := downloadSliceSpamCheckMap.Load(key); ok {
		rsp := &protos.RspDownloadSlice{
			Result: &protos.Result{
				State: protos.ResultState_RES_FAIL,
				Msg:   "repeated download slice request, refused",
			},
		}
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspDownloadSlice)
		return
	} else {
		var a any
		downloadSliceSpamCheckMap.Store(key, a)
	}

	go splitSendDownloadSliceData(ctx, &target, conn)
}

func splitSendDownloadSliceData(ctx context.Context, target *protos.ReqDownloadSlice, conn core.WriteCloser) {
	var rsp *protos.RspDownloadSlice
	var data [][]byte
	var slice *protos.DownloadSliceInfo
	for _, slice = range target.RspFileStorageInfo.SliceInfo {
		if slice.SliceNumber == target.SliceNumber {
			break
		}
	}
	sliceTaskId := slice.TaskId

	rsp, data = requests.RspDownloadSliceData(ctx, target, slice)
	if rsp == nil && data == nil {
		return
	}

	setWriteHookForRspDownloadSlice(conn)
	if task.DownloadSliceTaskMap.HashKey(sliceTaskId + slice.SliceStorageInfo.SliceHash) {
		rsp.Data = nil
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "duplicate request for the same slice in the same download task"
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspDownloadSlice)
		for _, buffer := range data {
			utils.ReleaseBuffer(buffer)
		}
		return
	}

	if !verifyDownloadSliceHash(target.RspFileStorageInfo.FileHash, target.SliceNumber, slice, data) {
		rsp.Data = nil
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "slice hash validation failed"
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspDownloadSlice)
		for _, buffer := range data {
			utils.ReleaseBuffer(buffer)
		}
		return
	}

	if rsp.SliceSize == 0 {
		utils.DebugLog("cannot find slice, sliceHash: ", slice.SliceStorageInfo.SliceHash)
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = LOSE_SLICE_MSG
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspDownloadSlice)
		for _, buffer := range data {
			utils.ReleaseBuffer(buffer)
		}
		return
	}

	utils.DebugLog("splitSendDownloadSliceData reqID =========", core.GetReqIdFromContext(ctx))
	sliceLen := rsp.SliceSize
	utils.DebugLog("sliceLen=========", sliceLen)
	dataStart := uint64(0)
	dataEnd := uint64(setting.MaxData)
	offsetStart := rsp.SliceInfo.SliceOffset.SliceOffsetStart
	offsetEnd := rsp.SliceInfo.SliceOffset.SliceOffsetStart + dataEnd

	tkSliceUID := rsp.TaskId + rsp.SliceInfo.SliceHash

	// save rsp for further report to SP
	downloadRspMap.Store(tkSliceUID, QueuedDownloadReportToSP{
		context:  ctx,
		response: rsp,
	})

	for _, packet := range data {
		utils.DebugLog("_____________________________")
		utils.DebugLog(dataStart, dataEnd, offsetStart, offsetEnd)

		_, newCtx := prepareSendDownloadSliceData(ctx, rsp, tkSliceUID)

		if dataEnd < sliceLen {
			utils.DebugLog("reqID-"+strconv.FormatUint(dataStart, 10)+" =========", strconv.FormatInt(core.GetReqIdFromContext(newCtx), 10))
			_ = p2pserver.GetP2pServer(ctx).SendMessage(
				newCtx,
				conn,
				requests.RspDownloadSliceDataSplit(rsp, dataStart, dataEnd, offsetStart, offsetEnd, rsp.SliceInfo.SliceOffset.SliceOffsetStart, rsp.SliceInfo.SliceOffset.SliceOffsetEnd, packet, false),
				header.RspDownloadSlice,
			)
			dataStart += setting.MaxData
			dataEnd += setting.MaxData
			offsetStart += setting.MaxData
			offsetEnd += setting.MaxData
		} else {
			utils.DebugLog("reqID-"+strconv.FormatUint(dataStart, 10)+" =========", strconv.FormatInt(core.GetReqIdFromContext(newCtx), 10))
			_ = p2pserver.GetP2pServer(ctx).SendMessage(
				newCtx,
				conn,
				requests.RspDownloadSliceDataSplit(rsp, dataStart, 0, offsetStart, 0, rsp.SliceInfo.SliceOffset.SliceOffsetStart, rsp.SliceInfo.SliceOffset.SliceOffsetEnd, packet, true),
				header.RspDownloadSlice)
			return
		}
	}
}

func prepareSendDownloadSliceData(ctx context.Context, rsp *protos.RspDownloadSlice, tkSliceUID string) (int64, context.Context) {
	packetId, newCtx := p2pserver.CreateNewContextPacketId(ctx)
	tkSlice := TaskSlice{
		TkSliceUID: tkSliceUID,
		SliceType:  SliceDownload,
	}
	PacketIdMap.Store(packetId, tkSlice)
	utils.DebugLogf("PacketIdMap.Store <==(%v, %v)", packetId, tkSlice)
	ctStat := DownSendCostTimeMap.StartSendPacket(rsp.TaskId + rsp.SliceInfo.SliceHash)
	utils.DebugLogf("downSendPacketWgMap.Store <== K:%v, V:%v]", tkSliceUID, ctStat)
	return packetId, newCtx
}

// RspDownloadSlice storagePP-PP
func RspDownloadSlice(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("RspDownloadSlice reqID =========", core.GetReqIdFromContext(ctx))
	costTime := core.GetRecvCostTimeFromContext(ctx)
	utils.DebugLog("get RspDownloadSlice, cost time: ", costTime)
	var target protos.RspDownloadSlice
	if err := VerifyMessage(ctx, header.RspDownloadSlice, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	defer utils.ReleaseBuffer(target.Data)

	fileReqId, found := getFileReqIdFromContext(ctx)
	if !found {
		utils.DebugLog("Can't find who created slice request", core.GetRemoteReqId(ctx))
		return
	}

	dTask, ok := task.GetDownloadTask(target.FileHash + target.WalletAddress + fileReqId)
	if !ok {
		utils.DebugLog("current task is stopped！！！！！！！！！！！！！！！！！！！！！！！！！！")
		return
	}

	if target.SliceSize <= 0 || (target.Result.State == protos.ResultState_RES_FAIL && target.Result.Msg == LOSE_SLICE_MSG) {
		pp.DebugLog(ctx, "slice was not found, will send msg to sp for retry, sliceHash: ", target.SliceInfo.SliceHash)
		setDownloadSliceFail(ctx, target.SliceInfo.SliceHash, target.TaskId, fileReqId, dTask)
		return
	}

	if target.Result.State == protos.ResultState_RES_FAIL {
		pp.ErrorLog(ctx, target.Result.Msg)
		return
	}

	// add up costTime
	tkSlice := target.TaskId + target.SliceInfo.SliceHash
	_ = DownRecvCostTimeMap.AddCostTime(tkSlice, costTime)

	if f, ok := task.DownloadFileMap.Load(target.FileHash + fileReqId); ok {
		fInfo := f.(*protos.RspFileStorageInfo)
		utils.DebugLog("get a slice -------")
		utils.DebugLog("SliceHash", target.SliceInfo.SliceHash)
		utils.DebugLog("SliceOffset", target.SliceInfo.SliceOffset)
		utils.DebugLog("length", len(target.Data))
		utils.DebugLog("sliceSize", target.SliceSize)
		if fInfo.EncryptionTag != "" {
			receiveSliceAndProgressEncrypted(ctx, &target, fInfo, dTask, costTime)
		} else {
			receiveSliceAndProgress(ctx, &target, fInfo, dTask, costTime)
		}
	} else {
		utils.DebugLogf("Received a slice from an outdated download request[FileHash=%v], ignoring... ", target.FileHash)
	}
}

func receiveSliceAndProgress(ctx context.Context, target *protos.RspDownloadSlice, fInfo *protos.RspFileStorageInfo,
	dTask *task.DownloadTask, costTime int64) {

	if success, ok := dTask.SuccessSlice[target.SliceInfo.SliceHash]; ok && success {
		utils.DebugLogf("Slice[%v] of file[%v] already received, skipping duplicate data,", target.SliceInfo.SliceHash, target.FileHash)
		return
	}
	err := task.SaveDownloadFile(ctx, target, fInfo)
	if err != nil {
		utils.ErrorLog("Download failed, failed saving file ", err)
		file.CloseDownloadSession(fInfo.FileHash + fInfo.ReqId)
		task.CleanDownloadFileAndConnMap(ctx, fInfo.FileHash, fInfo.ReqId)
		return
	}
	dataLen := uint64(len(target.Data))
	if s, ok := task.DownloadSliceProgress.Load(target.TaskId + target.SliceInfo.SliceHash + fInfo.ReqId); ok {
		alreadySize := s.(uint64)
		alreadySize += dataLen
		if alreadySize == target.SliceSize {
			utils.DebugLog("slice download finished", target.SliceInfo.SliceHash)
			task.DownloadSliceProgress.Delete(target.TaskId + target.SliceInfo.SliceHash + fInfo.ReqId)
			receivedSlice(ctx, target, fInfo, dTask)
		} else {
			task.DownloadSliceProgress.Store(target.TaskId+target.SliceInfo.SliceHash+fInfo.ReqId, alreadySize)
		}
	} else {
		// if data is sent at once
		if target.SliceSize == dataLen {
			receivedSlice(ctx, target, fInfo, dTask)
		} else {
			task.DownloadSliceProgress.Store(target.TaskId+target.SliceInfo.SliceHash+fInfo.ReqId, dataLen)
		}
	}
}

func receiveSliceAndProgressEncrypted(ctx context.Context, target *protos.RspDownloadSlice,
	fInfo *protos.RspFileStorageInfo, dTask *task.DownloadTask, costTime int64) {

	if success, ok := dTask.SuccessSlice[target.SliceInfo.SliceHash]; ok && success {
		utils.DebugLogf("Slice[%v] of file[%v] already received, skipping duplicate data,", target.SliceInfo.SliceHash, target.FileHash)
		return
	}

	dataToDecrypt := target.Data
	dataToDecryptSize := uint64(len(dataToDecrypt))
	encryptedOffset := target.SliceInfo.EncryptedSliceOffset

	if existingSlice, ok := task.DownloadEncryptedSlices.Load(target.SliceInfo.SliceHash + fInfo.ReqId); ok {
		existingSliceData := existingSlice.([]byte)
		copy(existingSliceData[encryptedOffset.SliceOffsetStart:encryptedOffset.SliceOffsetEnd], dataToDecrypt)
		dataToDecrypt = existingSliceData

		if s, ok := task.DownloadSliceProgress.Load(target.TaskId + target.SliceInfo.SliceHash + fInfo.ReqId); ok {
			existingSize := s.(uint64)
			dataToDecryptSize += existingSize
		}
	}

	if dataToDecryptSize >= target.SliceSize {
		// Decrypt slice data and save it to file
		decryptedData, err := decryptSliceData(dataToDecrypt)
		if err != nil {
			pp.ErrorLog(ctx, "Couldn't decrypt slice", err)
			return
		}
		target.Data = decryptedData
		err = task.SaveDownloadFile(ctx, target, fInfo)
		if err != nil {
			pp.ErrorLog(ctx, "Failed saving download file", err.Error())
			return
		}
		pp.DebugLog(ctx, "slice download finished", target.SliceInfo.SliceHash)
		task.DownloadSliceProgress.Delete(target.TaskId + target.SliceInfo.SliceHash + fInfo.ReqId)
		task.DownloadEncryptedSlices.Delete(target.SliceInfo.SliceHash + fInfo.ReqId)
		utils.DebugLog("slice task has been deleted...")
		receivedSlice(ctx, target, fInfo, dTask)
	} else {
		// Store partial slice data to memory
		dataToStore := dataToDecrypt
		if uint64(len(dataToStore)) < target.SliceSize {
			dataToStore = make([]byte, target.SliceSize)
			copy(dataToStore[encryptedOffset.SliceOffsetStart:encryptedOffset.SliceOffsetEnd], dataToDecrypt)
		}
		task.DownloadEncryptedSlices.Store(target.SliceInfo.SliceHash+fInfo.ReqId, dataToStore)
		task.DownloadSliceProgress.Store(target.TaskId+target.SliceInfo.SliceHash+fInfo.ReqId, dataToDecryptSize)
	}
}

func receivedSlice(ctx context.Context, target *protos.RspDownloadSlice, fInfo *protos.RspFileStorageInfo, dTask *task.DownloadTask) {
	file.SaveDownloadProgress(ctx, target.SliceInfo.SliceHash, fInfo.FileName, target.FileHash, target.SavePath, fInfo.ReqId)
	task.CleanDownloadTask(ctx, target.FileHash, target.SliceInfo.SliceHash, target.WalletAddress, fInfo.ReqId)
	task.DownloadProgress(ctx, target.FileHash, fInfo.ReqId, target.SliceSize)

	target.Result = &protos.Result{
		State: protos.ResultState_RES_SUCCESS,
	}
	setDownloadSliceSuccess(ctx, target.SliceInfo.SliceHash, dTask)
	// get total costTime
	tkSlice := target.TaskId + target.SliceInfo.SliceHash
	totalCostTime := DownRecvCostTimeMap.AddCostTime(tkSlice, int64(0))
	reportReq := SendReportDownloadResult(ctx, target, totalCostTime, false)
	metrics.StoredSliceCount.WithLabelValues("download").Inc()
	instantInboundSpeed := float64(target.SliceSize) / math.Max(float64(totalCostTime), 1)
	metrics.InboundSpeed.WithLabelValues(reportReq.OpponentP2PAddress).Set(instantInboundSpeed)
	DownRecvCostTimeMap.DeleteRecord(tkSlice)
}

// SendReportDownloadResult  PP-SP OR StoragePP-SP
func SendReportDownloadResult(ctx context.Context, target *protos.RspDownloadSlice, costTime int64, isPP bool) *protos.ReqReportDownloadResult {
	utils.DebugLog("ReportDownloadResult report result target.fileHash = ", target.FileHash)
	req := requests.ReqReportDownloadResultData(ctx, target, costTime, isPP)
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, req, header.ReqReportDownloadResult)
	return req
}

// SendReportStreamingResult  P-SP OR PP-SP
func SendReportStreamingResult(ctx context.Context, target *protos.RspDownloadSlice, isPP bool) {
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqReportStreamResultData(ctx, target, isPP), header.ReqReportDownloadResult)
}

func DownloadFileSlices(ctx context.Context, target *protos.RspFileStorageInfo, reqId string) {
	utils.DebugLog("DownloadFileSlice(&target)", target)
	fileSize := uint64(0)
	dTask, _ := task.GetDownloadTask(target.FileHash + target.WalletAddress + reqId)
	for _, sliceInfo := range target.SliceInfo {
		fileSize += sliceInfo.SliceOffset.SliceOffsetEnd - sliceInfo.SliceOffset.SliceOffsetStart
	}
	utils.DebugLogf("file size: %v  raw file size: %v\n", fileSize, target.FileSize)

	sp := &task.DownloadSP{
		RawSize:        int64(target.FileSize),
		TotalSize:      int64(fileSize),
		DownloadedSize: 0,
	}
	if !file.CheckFileExisting(ctx, target.FileHash, target.FileName, target.SavePath, target.EncryptionTag, reqId) {
		pp.Log(ctx, "download starts: ")
		task.DownloadSpeedOfProgress.Store(target.FileHash+reqId, sp)
		for _, slice := range target.SliceInfo {
			var re string
			if file.CheckSliceExisting(target.FileHash, target.FileName, slice.SliceStorageInfo.SliceHash, reqId) {
				re = "slice exists"
				task.DownloadProgress(ctx, target.FileHash, reqId, slice.SliceOffset.SliceOffsetEnd-slice.SliceOffset.SliceOffsetStart)
				task.CleanDownloadTask(ctx, target.FileHash, slice.SliceStorageInfo.SliceHash, target.WalletAddress, reqId)
				setDownloadSliceSuccess(ctx, slice.SliceStorageInfo.SliceHash, dTask)
			} else {
				re = "request for slice data sent"
				req := requests.ReqDownloadSliceData(ctx, target, slice)
				newCtx := createAndRegisterSliceReqId(ctx, reqId)
				SendReqDownloadSlice(newCtx, target.FileHash, slice, req, reqId)
			}
			utils.DebugLog("slice info ======= \ntaskid: ", slice.TaskId,
				"\nslicehash: ", slice.SliceStorageInfo.SliceHash,
				"\nslicenumber: ", slice.SliceNumber, "\n result:", re)
		}
	} else {
		task.DownloadResult(ctx, target.FileHash, false, "file exists already.")
		task.DeleteDownloadTask(target.FileHash, target.WalletAddress, target.ReqId)
	}
}

func SendReqDownloadSlice(ctx context.Context, fileHash string, sliceInfo *protos.DownloadSliceInfo, req *protos.ReqDownloadSlice, fileReqId string) {
	utils.DebugLog("req = ", req)

	networkAddress := sliceInfo.StoragePpInfo.NetworkAddress
	key := "download#" + fileHash + sliceInfo.StoragePpInfo.P2PAddress + fileReqId
	metrics.UploadPerformanceLogNow(fileHash + ":SND_REQ_SLICE_DATA:" + strconv.FormatInt(int64(sliceInfo.SliceOffset.SliceOffsetStart+(req.SliceNumber-1)*setting.MaxSliceSize), 10) + ":" + networkAddress)
	err := p2pserver.GetP2pServer(ctx).SendMessageByCachedConn(ctx, key, networkAddress, req, header.ReqDownloadSlice, nil)
	if err != nil {
		pp.ErrorLogf(ctx, "Failed to create connection with %v: %v", networkAddress, utils.FormatError(err))
		if dTask, ok := task.GetDownloadTask(fileHash + req.RspFileStorageInfo.WalletAddress + fileReqId); ok {
			setDownloadSliceFail(ctx, sliceInfo.SliceStorageInfo.SliceHash, req.RspFileStorageInfo.TaskId, fileReqId, dTask)
		}
	}
}

// RspReportDownloadResult  SP-P OR SP-PP
func RspReportDownloadResult(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get RspReportDownloadResult")
	var target protos.RspReportDownloadResult
	if err := VerifyMessage(ctx, header.RspReportDownloadResult, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if requests.UnmarshalData(ctx, &target) {
		utils.DebugLog("result", target.Result.State, target.Result.Msg)
	}
}

func DownloadSlicePause(ctx context.Context, fileHash, reqID string) {
	if setting.CheckLogin() {
		// storeResponseWriter(reqID, w)
		task.DownloadTaskMap.Delete(fileHash + setting.WalletAddress + task.LOCAL_REQID)
		task.CleanDownloadFileAndConnMap(ctx, fileHash, reqID)
	}
}

func DownloadSliceCancel(ctx context.Context, fileHash, reqID string) {
	if setting.CheckLogin() {
		task.DownloadTaskMap.Delete(fileHash + setting.WalletAddress + task.LOCAL_REQID)
		task.CleanDownloadFileAndConnMap(ctx, fileHash, reqID)
		task.CancelDownloadTask(fileHash)
	}
}

func decryptSliceData(dataToDecrypt []byte) ([]byte, error) {
	encryptedSlice := protos.EncryptedSlice{}
	err := proto.Unmarshal(dataToDecrypt, &encryptedSlice)
	if err != nil {
		utils.ErrorLog("Couldn't unmarshal protobuf to encrypted slice", err)
		return nil, err
	}

	key, err := hdkey.MasterKeyForSliceEncryption(setting.WalletPrivateKey.Bytes(), encryptedSlice.HdkeyNonce)
	if err != nil {
		utils.ErrorLog("Couldn't generate slice encryption master key", err)
		return nil, err
	}

	return encryption.DecryptAES(key.PrivateKey(), encryptedSlice.Data, encryptedSlice.AesNonce, false)
}

func verifyDownloadSliceHash(fileHash string, sliceNumber uint64, slice *protos.DownloadSliceInfo, buffers [][]byte) bool {
	var data []byte
	for _, buffer := range buffers {
		data = append(data, buffer...)
	}
	sliceHash, err := crypto.CalcSliceHash(data, fileHash, sliceNumber)
	if err != nil {
		utils.ErrorLog(err)
		return false
	}

	return slice.SliceStorageInfo.SliceHash == sliceHash
}

func setDownloadSliceSuccess(ctx context.Context, sliceHash string, dTask *task.DownloadTask) {
	dTask.SetSliceSuccess(sliceHash)
	CheckAndSendRetryMessage(ctx, dTask)
}

func setDownloadSliceFail(ctx context.Context, sliceHash, taskId, fileReqId string, dTask *task.DownloadTask) {
	dTask.AddFailedSlice(sliceHash)
	CheckAndSendRetryMessage(ctx, dTask)
}

func handleDownloadSend(tkSlice TaskSlice, costTime int64) {
	var newCostTimeStat CostTimeStat
	isDownloadFinished := false
	newCostTimeStat = DownSendCostTimeMap.FinishSendPacket(tkSlice.TkSliceUID, costTime)
	isDownloadFinished = newCostTimeStat.PacketCount == 0 && newCostTimeStat.TotalCostTime > 0
	if isDownloadFinished {
		if val, ok := downloadRspMap.Load(tkSlice.TkSliceUID); ok {
			queuedReport := val.(QueuedDownloadReportToSP)
			// report download slice result
			reportReq := SendReportDownloadResult(queuedReport.context, queuedReport.response, newCostTimeStat.TotalCostTime, true)
			instantOutboundSpeed := float64(queuedReport.response.SliceSize) / math.Max(float64(newCostTimeStat.TotalCostTime), 1)
			metrics.OutboundSpeed.WithLabelValues(reportReq.OpponentP2PAddress).Set(instantOutboundSpeed)
			// set task status as finished
			task.DownloadSliceTaskMap.Store(tkSlice.TkSliceUID, true)
			downloadRspMap.Delete(tkSlice.TkSliceUID)
			DownSendCostTimeMap.DeleteRecord(tkSlice.TkSliceUID)
		}
	}
}

func createAndRegisterSliceReqId(ctx context.Context, fileReqId string) context.Context {
	newReqId, _ := utils.NextSnowFlakeId()
	core.InheritRpcLoggerFromParentReqId(ctx, newReqId)
	sliceReqId := uuid.New().String()
	newCtx := core.CreateContextWithReqId(ctx, newReqId)
	core.StoreRemoteReqId(newReqId, sliceReqId)
	task.SliceSessionMap.Store(sliceReqId, fileReqId)
	return newCtx
}

func getFileReqIdFromContext(ctx context.Context) (string, bool) {
	remoteReqId := core.GetRemoteReqId(ctx)
	if remoteReqId == "" {
		return "", false
	}
	fileReqId, found := task.SliceSessionMap.Load(remoteReqId)
	if !found {
		return remoteReqId, true
	}
	return fileReqId.(string), true
}
