package event

// Author j cc
import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/metrics"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/utils/types"
	"google.golang.org/protobuf/proto"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/encryption"
	"github.com/stratosnet/sds/utils/encryption/hdkey"
)

const (
	LOSE_SLICE_MSG = "cannot find the file slice"
)

var (
	downloadRspMap      = &sync.Map{} // K: tkId+sliceHash, V: *QueuedDownloadReportToSP
	downSendCostTimeMap = &downSendCostTime{
		dataMap: utils.NewAutoCleanUnsafeMap(30 * time.Minute), // make(map[string]*CostTimeStat), // K: tkId+sliceHash, V: CostTimeStat{TotalCostTime, PacketCount}
		mux:     sync.Mutex{},
	}
	downRecvCostTimeMap = &downRecvCostTime{
		dataMap: utils.NewAutoCleanUnsafeMap(30 * time.Minute), // make(map[string]int64), // K: tkId+sliceHash, V: CostTimeStat{TotalCostTime, PacketCount}
		mux:     sync.Mutex{},
	}
)

type downSendCostTime struct {
	dataMap *utils.AutoCleanUnsafeMap // map[string]*CostTimeStat // K: tkId+sliceHash, V: CostTimeStat{TotalCostTime, PacketCount}
	mux     sync.Mutex
}
type downRecvCostTime struct {
	dataMap *utils.AutoCleanUnsafeMap // map[string]int64 // K: tkId+sliceHash, V: CostTimeStat{TotalCostTime, PacketCount}
	mux     sync.Mutex
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
			Message: header.RspDownloadSlice,
			Fn:      HandleSendPacketCostTime,
		}
		var hooks []core.WriteHook
		hooks = append(hooks, hook)
		conn.SetWriteHook(hooks)
	case *cf.ClientConn:
		hook := cf.WriteHook{
			Message: header.RspDownloadSlice,
			Fn:      HandleSendPacketCostTime,
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
	if requests.UnmarshalData(ctx, &target) {
		rsp := requests.RspDownloadSliceData(&target)
		setWriteHookForRspDownloadSlice(conn)
		if task.DownloadSliceTaskMap.HashKey(target.TaskId + target.SliceInfo.SliceHash) {
			rsp.Data = nil
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "duplicate request for the same slice in the same download task"
			_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspDownloadSlice)
			return
		}

		if target.PpNodeSign == nil || target.SpNodeSign == nil {
			rsp.Data = nil
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "empty signature"
			_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspDownloadSlice)
			return
		}

		if !verifyDownloadSliceSign(&target, rsp) {
			rsp.Data = nil
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "signature validation failed"
			_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspDownloadSlice)
			return
		}

		if rsp.SliceSize == 0 {
			utils.DebugLog("cannot find slice, sliceHash: ", target.SliceInfo.SliceHash)
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = LOSE_SLICE_MSG
			_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspDownloadSlice)
			return
		}

		splitSendDownloadSliceData(ctx, rsp, conn)

		//SendReportDownloadResult(ctx, rsp, end.Sub(start).Milliseconds(), true)

		//task.DownloadSliceTaskMap.Store(target.TaskId+target.SliceInfo.SliceHash, true)
	}
}

func splitSendDownloadSliceData(ctx context.Context, rsp *protos.RspDownloadSlice, conn core.WriteCloser) {
	utils.DebugLog("splitSendDownloadSliceData reqID =========", core.GetReqIdFromContext(ctx))
	dataLen := uint64(len(rsp.Data))
	utils.DebugLog("dataLen=========", dataLen)
	dataStart := uint64(0)
	dataEnd := uint64(setting.MAXDATA)
	offsetStart := rsp.SliceInfo.SliceOffset.SliceOffsetStart
	offsetEnd := rsp.SliceInfo.SliceOffset.SliceOffsetStart + dataEnd

	tkSliceUID := rsp.TaskId + rsp.SliceInfo.SliceHash

	// save rsp for further report to SP
	downloadRspMap.Store(tkSliceUID, QueuedDownloadReportToSP{
		context:  ctx,
		response: rsp,
	})

	for {
		utils.DebugLog("_____________________________")
		utils.DebugLog(dataStart, dataEnd, offsetStart, offsetEnd)

		_, newCtx := prepareSendDownloadSliceData(ctx, rsp, tkSliceUID)

		if dataEnd < dataLen {
			utils.DebugLog("reqID-"+strconv.FormatUint(dataStart, 10)+" =========", strconv.FormatInt(core.GetReqIdFromContext(newCtx), 10))
			_ = p2pserver.GetP2pServer(ctx).SendMessage(newCtx, conn, requests.RspDownloadSliceDataSplit(rsp, dataStart, dataEnd, offsetStart, offsetEnd,
				rsp.SliceInfo.SliceOffset.SliceOffsetStart, rsp.SliceInfo.SliceOffset.SliceOffsetEnd, false), header.RspDownloadSlice)
			dataStart += setting.MAXDATA
			dataEnd += setting.MAXDATA
			offsetStart += setting.MAXDATA
			offsetEnd += setting.MAXDATA
		} else {
			utils.DebugLog("reqID-"+strconv.FormatUint(dataStart, 10)+" =========", strconv.FormatInt(core.GetReqIdFromContext(newCtx), 10))
			_ = p2pserver.GetP2pServer(ctx).SendMessage(newCtx, conn, requests.RspDownloadSliceDataSplit(rsp, dataStart, 0, offsetStart, 0,
				rsp.SliceInfo.SliceOffset.SliceOffsetStart, rsp.SliceInfo.SliceOffset.SliceOffsetEnd, true), header.RspDownloadSlice)
			return
		}
	}
}

func prepareSendDownloadSliceData(ctx context.Context, rsp *protos.RspDownloadSlice, tkSliceUID string) (int64, context.Context) {
	packetId, newCtx := p2pserver.CreateNewContextPacketId(ctx)
	tkSlice := TaskSlice{
		TkSliceUID: tkSliceUID,
		IsUpload:   false,
	}
	PacketIdMap.Store(packetId, tkSlice)
	utils.DebugLogf("PacketIdMap.Store <==(%v, %v)", packetId, tkSlice)
	downSendCostTimeMap.mux.Lock()
	var ctStat = CostTimeStat{}
	if val, ok := downSendCostTimeMap.dataMap.Load(rsp.TaskId + rsp.SliceInfo.SliceHash); ok {
		ctStat = val.(CostTimeStat)
	}
	ctStat.PacketCount = ctStat.PacketCount + 1
	downSendCostTimeMap.dataMap.Store(tkSliceUID, ctStat)
	downSendCostTimeMap.mux.Unlock()
	utils.DebugLogf("downSendPacketWgMap.Store <== K:%v, V:%v]", tkSliceUID, ctStat)
	return packetId, newCtx
}

// RspDownloadSlice storagePP-PP
func RspDownloadSlice(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("RspDownloadSlice reqID =========", core.GetReqIdFromContext(ctx))
	costTime := core.GetRecvCostTimeFromContext(ctx)
	pp.DebugLog(ctx, "get RspDownloadSlice, cost time: ", costTime)
	var target protos.RspDownloadSlice
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	fileReqId, found := getFileReqIdFromContext(ctx)
	if !found {
		utils.DebugLog("Can't find who created slice request", core.GetRemoteReqId(ctx))
		return
	}

	dTask, ok := task.GetDownloadTask(target.FileHash, target.WalletAddress, fileReqId)
	if !ok {
		pp.DebugLog(ctx, "current task is stopped！！！！！！！！！！！！！！！！！！！！！！！！！！")
		return
	}

	if target.SliceSize <= 0 || (target.Result.State == protos.ResultState_RES_FAIL && target.Result.Msg == LOSE_SLICE_MSG) {
		pp.DebugLog(ctx, "slice was not found, will send msg to sp for retry, sliceHash: ", target.SliceInfo.SliceHash)
		setDownloadSliceFail(ctx, target.SliceInfo.SliceHash, target.TaskId, target.IsVideoCaching, dTask)
		return
	}

	if target.Result.State == protos.ResultState_RES_FAIL {
		pp.ErrorLog(ctx, target.Result.Msg)
		return
	}

	// add up costTime
	totalCostTime := costTime
	tkSlice := target.TaskId + target.SliceInfo.SliceHash
	downRecvCostTimeMap.mux.Lock()
	if val, ok := downRecvCostTimeMap.dataMap.Load(tkSlice); ok {
		totalCostTime += val.(int64)
	}
	downRecvCostTimeMap.dataMap.Store(tkSlice, totalCostTime)
	downRecvCostTimeMap.mux.Unlock()

	if f, ok := task.DownloadFileMap.Load(target.FileHash + fileReqId); ok {
		fInfo := f.(*protos.RspFileStorageInfo)
		pp.DebugLog(ctx, "get a slice -------")
		pp.DebugLog(ctx, "SliceHash", target.SliceInfo.SliceHash)
		pp.DebugLog(ctx, "SliceOffset", target.SliceInfo.SliceOffset)
		pp.DebugLog(ctx, "length", len(target.Data))
		pp.DebugLog(ctx, "sliceSize", target.SliceSize)
		if fInfo.EncryptionTag != "" {
			receiveSliceAndProgressEncrypted(ctx, &target, fInfo, dTask, costTime)
		} else {
			receiveSliceAndProgress(ctx, &target, fInfo, dTask, costTime)
		}
		if !fInfo.IsVideoStream {
			task.DownloadProgress(ctx, target.FileHash, fileReqId, uint64(len(target.Data)))
		}
	} else {
		utils.DebugLog("DownloadFileMap doesn't have entry with file hash", target.FileHash)
	}
}

func receiveSliceAndProgress(ctx context.Context, target *protos.RspDownloadSlice, fInfo *protos.RspFileStorageInfo,
	dTask *task.DownloadTask, costTime int64) {
	if task.SaveDownloadFile(ctx, target, fInfo) {
		dataLen := uint64(len(target.Data))
		if s, ok := task.DownloadSliceProgress.Load(target.SliceInfo.SliceHash + fInfo.ReqId); ok {
			alreadySize := s.(uint64)
			alreadySize += dataLen
			if alreadySize == target.SliceSize {
				pp.DebugLog(ctx, "slice download finished", target.SliceInfo.SliceHash)
				task.DownloadSliceProgress.Delete(target.SliceInfo.SliceHash + fInfo.ReqId)
				receivedSlice(ctx, target, fInfo, dTask)
			} else {
				task.DownloadSliceProgress.Store(target.SliceInfo.SliceHash+fInfo.ReqId, alreadySize)
			}
		} else {
			// if data is sent at once
			if target.SliceSize == dataLen {
				receivedSlice(ctx, target, fInfo, dTask)
			} else {
				task.DownloadSliceProgress.Store(target.SliceInfo.SliceHash+fInfo.ReqId, dataLen)
			}
		}
	} else {
		utils.DebugLog("Download failed: not able to write to the target file.")
		file.CloseDownloadSession(fInfo.FileHash + fInfo.ReqId)
		task.CleanDownloadFileAndConnMap(ctx, fInfo.FileHash, fInfo.ReqId)
	}
}

func receiveSliceAndProgressEncrypted(ctx context.Context, target *protos.RspDownloadSlice,
	fInfo *protos.RspFileStorageInfo, dTask *task.DownloadTask, costTime int64) {
	dataToDecrypt := target.Data
	dataToDecryptSize := uint64(len(dataToDecrypt))
	encryptedOffset := target.SliceInfo.EncryptedSliceOffset

	if existingSlice, ok := task.DownloadEncryptedSlices.Load(target.SliceInfo.SliceHash + fInfo.ReqId); ok {
		existingSliceData := existingSlice.([]byte)
		copy(existingSliceData[encryptedOffset.SliceOffsetStart:encryptedOffset.SliceOffsetEnd], dataToDecrypt)
		dataToDecrypt = existingSliceData

		if s, ok := task.DownloadSliceProgress.Load(target.SliceInfo.SliceHash + fInfo.ReqId); ok {
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

		if task.SaveDownloadFile(ctx, target, fInfo) {
			pp.DebugLog(ctx, "slice download finished", target.SliceInfo.SliceHash)
			task.DownloadSliceProgress.Delete(target.SliceInfo.SliceHash + fInfo.ReqId)
			task.DownloadEncryptedSlices.Delete(target.SliceInfo.SliceHash + fInfo.ReqId)
			receivedSlice(ctx, target, fInfo, dTask)
		}
	} else {
		// Store partial slice data to memory
		dataToStore := dataToDecrypt
		if uint64(len(dataToStore)) < target.SliceSize {
			dataToStore = make([]byte, target.SliceSize)
			copy(dataToStore[encryptedOffset.SliceOffsetStart:encryptedOffset.SliceOffsetEnd], dataToDecrypt)
		}
		task.DownloadEncryptedSlices.Store(target.SliceInfo.SliceHash+fInfo.ReqId, dataToStore)
		task.DownloadSliceProgress.Store(target.SliceInfo.SliceHash+fInfo.ReqId, dataToDecryptSize)
	}
}

func receivedSlice(ctx context.Context, target *protos.RspDownloadSlice, fInfo *protos.RspFileStorageInfo, dTask *task.DownloadTask) {
	file.SaveDownloadProgress(ctx, target.SliceInfo.SliceHash, fInfo.FileName, target.FileHash, target.SavePath, fInfo.ReqId)
	task.CleanDownloadTask(ctx, target.FileHash, target.SliceInfo.SliceHash, target.WalletAddress, fInfo.ReqId)
	target.Result = &protos.Result{
		State: protos.ResultState_RES_SUCCESS,
	}
	if fInfo.IsVideoStream && !target.IsVideoCaching {
		putData(ctx, HTTPDownloadSlice, target)
	} else if fInfo.IsVideoStream && target.IsVideoCaching {
		videoCacheKeep(fInfo.FileHash, target.TaskId)
	}
	setDownloadSliceSuccess(ctx, target.SliceInfo.SliceHash, dTask)
	// get total costTime
	totalCostTime := int64(0)
	tkSlice := target.TaskId + target.SliceInfo.SliceHash
	downRecvCostTimeMap.mux.Lock()
	defer downRecvCostTimeMap.mux.Unlock()
	if val, ok := downRecvCostTimeMap.dataMap.Load(tkSlice); ok {
		totalCostTime += val.(int64)
	}
	reportReq := SendReportDownloadResult(ctx, target, totalCostTime, false)
	metrics.StoredSliceCount.WithLabelValues("download").Inc()
	instantInboundSpeed := float64(target.SliceSize) / math.Max(float64(totalCostTime), 1)
	metrics.InboundSpeed.WithLabelValues(reportReq.OpponentP2PAddress).Set(instantInboundSpeed)
	downRecvCostTimeMap.dataMap.Delete(tkSlice)
}

func videoCacheKeep(fileHash, taskID string) {
	utils.DebugLogf("download keep fileHash = %v  taskID = %v", fileHash, taskID)
	if ing, ok := task.VideoCacheTaskMap.Load(fileHash); ok {
		ING := ing.(*task.VideoCacheTask)
		ING.DownloadCh <- true
	}
}

// SendReportDownloadResult  PP-SP OR StoragePP-SP
func SendReportDownloadResult(ctx context.Context, target *protos.RspDownloadSlice, costTime int64, isPP bool) *protos.ReqReportDownloadResult {
	pp.DebugLog(ctx, "ReportDownloadResult report result target.fileHash = ", target.FileHash)
	req := requests.ReqReportDownloadResultData(target, costTime, isPP)
	p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, req, header.ReqReportDownloadResult)
	return req
}

// SendReportStreamingResult  P-SP OR PP-SP
func SendReportStreamingResult(ctx context.Context, target *protos.RspDownloadSlice, isPP bool) {
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqReportStreamResultData(target, isPP), header.ReqReportDownloadResult)
}

func DownloadFileSlice(ctx context.Context, target *protos.RspFileStorageInfo) {
	pp.DebugLog(ctx, "DownloadFileSlice(&target)", target)
	fileSize := uint64(0)
	dTask, _ := task.GetDownloadTask(target.FileHash, target.WalletAddress, target.ReqId)
	for _, sliceInfo := range target.SliceInfo {
		fileSize += sliceInfo.SliceStorageInfo.SliceSize
	}
	pp.DebugLog(ctx, fmt.Sprintf("file size: %v  raw file size: %v\n", fileSize, target.FileSize))

	sp := &task.DownloadSP{
		RawSize:        int64(target.FileSize),
		TotalSize:      int64(fileSize),
		DownloadedSize: 0,
	}
	if !file.CheckFileExisting(ctx, target.FileHash, target.FileName, target.SavePath, target.EncryptionTag, target.ReqId) {
		pp.Log(ctx, "download starts: ")
		task.DownloadSpeedOfProgress.Store(target.FileHash+target.ReqId, sp)
		for _, rsp := range target.SliceInfo {
			pp.DebugLog(ctx, "taskid ======= ", rsp.TaskId)
			if file.CheckSliceExisting(target.FileHash, target.FileName, rsp.SliceStorageInfo.SliceHash, target.SavePath, target.ReqId) {
				pp.Log(ctx, "slice exist already,", rsp.SliceStorageInfo.SliceHash)
				task.DownloadProgress(ctx, target.FileHash, target.ReqId, rsp.SliceStorageInfo.SliceSize)
				task.CleanDownloadTask(ctx, target.FileHash, rsp.SliceStorageInfo.SliceHash, target.WalletAddress, target.ReqId)
				setDownloadSliceSuccess(ctx, rsp.SliceStorageInfo.SliceHash, dTask)
			} else {
				pp.DebugLog(ctx, "request download data")
				req := requests.ReqDownloadSliceData(target, rsp)
				newCtx := createAndRegisterSliceReqId(ctx, target.ReqId)
				SendReqDownloadSlice(newCtx, target.FileHash, rsp, req, target.ReqId)
			}
		}
	} else {
		pp.Log(ctx, "file exists already!")
		task.DeleteDownloadTask(target.FileHash, target.WalletAddress, target.ReqId)
	}
}

func SendReqDownloadSlice(ctx context.Context, fileHash string, sliceInfo *protos.DownloadSliceInfo, req *protos.ReqDownloadSlice, fileReqId string) {
	pp.DebugLog(ctx, "req = ", req)

	networkAddress := sliceInfo.StoragePpInfo.NetworkAddress
	key := "download#" + fileHash + sliceInfo.StoragePpInfo.P2PAddress + fileReqId
	metrics.UploadPerformanceLogNow(fileHash + ":SND_REQ_SLICE_DATA:" + strconv.FormatInt(int64(req.SliceInfo.SliceOffset.SliceOffsetStart+(req.SliceNumber-1)*33554432), 10) + ":" + networkAddress)
	err := p2pserver.GetP2pServer(ctx).SendMessageByCachedConn(ctx, key, networkAddress, req, header.ReqDownloadSlice, nil)
	if err != nil {
		pp.ErrorLogf(ctx, "Failed to create connection with %v: %v", networkAddress, utils.FormatError(err))
		if dTask, ok := task.GetDownloadTask(fileHash, req.WalletAddress, fileReqId); ok {
			setDownloadSliceFail(ctx, sliceInfo.SliceStorageInfo.SliceHash, req.TaskId, req.IsVideoCaching, dTask)
		}
	}
}

// RspReportDownloadResult  SP-P OR SP-PP
func RspReportDownloadResult(ctx context.Context, conn core.WriteCloser) {
	pp.DebugLog(ctx, "get RspReportDownloadResult")
	var target protos.RspReportDownloadResult
	if requests.UnmarshalData(ctx, &target) {
		pp.DebugLog(ctx, "result", target.Result.State, target.Result.Msg)
	}
}

func RspDownloadSliceWrong(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("RspDownloadSlice")
	var target protos.RspDownloadSliceWrong
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		return
	}

	utils.DebugLog("RspDownloadSliceWrong", target.NewSliceInfo.SliceStorageInfo.SliceHash)
	if dlTask, ok := task.DownloadTaskMap.Load(target.FileHash + target.WalletAddress + task.LOCAL_REQID); ok {
		downloadTask := dlTask.(*task.DownloadTask)
		if sInfo, ok := downloadTask.SliceInfo[target.NewSliceInfo.SliceStorageInfo.SliceHash]; ok {
			sInfo.StoragePpInfo.P2PAddress = target.NewSliceInfo.StoragePpInfo.P2PAddress
			sInfo.StoragePpInfo.WalletAddress = target.NewSliceInfo.StoragePpInfo.WalletAddress
			sInfo.StoragePpInfo.NetworkAddress = target.NewSliceInfo.StoragePpInfo.NetworkAddress
			_ = p2pserver.GetP2pServer(ctx).TransferSendMessageToPPServ(ctx, target.NewSliceInfo.StoragePpInfo.NetworkAddress, requests.RspDownloadSliceWrong(&target))
		}
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

	key, err := hdkey.MasterKeyForSliceEncryption(setting.WalletPrivateKey, encryptedSlice.HdkeyNonce)
	if err != nil {
		utils.ErrorLog("Couldn't generate slice encryption master key", err)
		return nil, err
	}

	return encryption.DecryptAES(key.PrivateKey(), encryptedSlice.Data, encryptedSlice.AesNonce)
}

func verifyDownloadSliceSign(target *protos.ReqDownloadSlice, rsp *protos.RspDownloadSlice) bool {
	// verify pp address
	if !types.VerifyP2pAddrBytes(target.PpP2PPubkey, target.P2PAddress) {
		utils.ErrorLogf("ppP2pPubkey validation failed, ppP2PAddress:[%v], ppP2PPubKey:[%v]", target.P2PAddress, target.PpP2PPubkey)
		return false
	}

	// verify node signature from the pp
	msg := utils.GetReqDownloadSlicePpNodeSignMessage(target.P2PAddress, setting.P2PAddress, target.SliceInfo.SliceHash, header.ReqDownloadSlice)
	if !types.VerifyP2pSignBytes(target.PpP2PPubkey, target.PpNodeSign, msg) {
		utils.ErrorLog("pp node signature validation failed, msg:", msg)
		return false
	}

	spP2pPubkey, err := requests.GetSpPubkey(target.SpP2PAddress)
	if err != nil {
		utils.ErrorLog("failed to find spP2pPubkey: ", err)
		return false
	}

	// verify sp address
	if !types.VerifyP2pAddrBytes(spP2pPubkey, target.SpP2PAddress) {
		utils.ErrorLogf("spP2pPubkey validation failed, spP2PAddress:[%v], spP2PPubKey:[%v]", target.SpP2PAddress, spP2pPubkey)
		return false
	}

	// verify sp node signature
	msg = utils.GetReqDownloadSliceSpNodeSignMessage(setting.P2PAddress, target.SpP2PAddress, target.SliceInfo.SliceHash, header.ReqDownloadSlice)
	if !types.VerifyP2pSignBytes(spP2pPubkey, target.SpNodeSign, msg) {
		utils.ErrorLog("sp node signature validation failed, msg: ", msg)
		return false
	}

	return target.SliceInfo.SliceHash == utils.CalcSliceHash(rsp.Data, target.FileHash, target.SliceNumber)
}

func setDownloadSliceSuccess(ctx context.Context, sliceHash string, dTask *task.DownloadTask) {
	dTask.SetSliceSuccess(sliceHash)
	CheckAndSendRetryMessage(ctx, dTask)
}

func setDownloadSliceFail(ctx context.Context, sliceHash, taskId string, isVideoCaching bool, dTask *task.DownloadTask) {
	dTask.AddFailedSlice(sliceHash)
	if isVideoCaching {
		videoCacheKeep(dTask.FileHash, taskId)
	}
	CheckAndSendRetryMessage(ctx, dTask)
}

func handleDownloadSend(tkSlice TaskSlice, costTime int64) {
	var newCostTimeStat = CostTimeStat{}
	isDownloadFinished := false
	downSendCostTimeMap.mux.Lock()
	if val, ok := downSendCostTimeMap.dataMap.Load(tkSlice.TkSliceUID); ok {
		oriCostTimeStat := val.(CostTimeStat)
		newCostTimeStat.TotalCostTime = costTime + oriCostTimeStat.TotalCostTime
		newCostTimeStat.PacketCount = oriCostTimeStat.PacketCount - 1
		// not counting if CostTimeStat not found from dataMap
		if newCostTimeStat.PacketCount >= 0 {
			downSendCostTimeMap.dataMap.Store(tkSlice.TkSliceUID, newCostTimeStat)
		}
		// return isDownloadFinished
		isDownloadFinished = newCostTimeStat.PacketCount == 0
	}
	downSendCostTimeMap.mux.Unlock()
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
