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
	UpSendCostTimeMap = &upSendCostTime{
		dataMap: utils.NewAutoCleanUnsafeMap(30 * time.Minute), // make(map[string]*CostTimeStat) // K: tkId+sliceNum, V: CostTimeStat{TotalCostTime, PacketCount}
		mux:     sync.Mutex{},
	}
	UpRecvCostTimeMap = &upRecvCostTime{
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
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	// check if signatures exist
	if target.SliceNumAddr.SpNodeSign == nil || target.PpNodeSign == nil {
		rsp := &protos.RspUploadFileSlice{
			Result: &protos.Result{
				State: protos.ResultState_RES_FAIL,
				Msg:   "missing signature(s)",
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
	if target.SliceNumAddr.PpInfo.P2PAddress != setting.P2PAddress {
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
	tkSlice := target.TaskId + strconv.FormatUint(target.SliceNumAddr.SliceNumber, 10)
	UpRecvCostTimeMap.mux.Lock()
	if val, ok := UpRecvCostTimeMap.dataMap.Load(tkSlice); ok {
		totalCostTime += val.(int64)
	}
	UpRecvCostTimeMap.dataMap.Store(tkSlice, totalCostTime)
	UpRecvCostTimeMap.mux.Unlock()
	timeEntry := time.Now().UnixMicro() - core.TimeRcv
	_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, requests.UploadSpeedOfProgressData(target.FileHash, uint64(len(target.Data)), (target.SliceNumAddr.SliceNumber-1)*33554432+target.SliceInfo.SliceOffset.SliceOffsetStart, timeEntry), header.UploadSpeedOfProgress)

	if err := task.SaveUploadFile(&target); err != nil {
		// save failed, not handling yet
		utils.ErrorLog("SaveUploadFile failed", err.Error())
		return
	}
	sliceSize, err := file.GetSliceSize(target.SliceInfo.SliceHash)
	if err != nil {
		utils.ErrorLog("Failed getting slice size", err.Error())
		return
	}
	utils.DebugLogf("ReqUploadFileSlice saving slice %v  current_size %v  total_size %v", target.SliceInfo.SliceHash, sliceSize, target.SliceSize)
	if sliceSize == int64(target.SliceSize) {
		utils.DebugLog("the slice upload finished", target.SliceInfo.SliceHash)
		// respond to PP in case the size is correct but actually not success
		sliceData, err := file.GetSliceData(target.SliceInfo.SliceHash)
		if err != nil {
			utils.ErrorLog("Failed getting slice data", err.Error())
			return
		}
		if utils.CalcSliceHash(sliceData, target.FileHash, target.SliceNumAddr.SliceNumber) == target.SliceInfo.SliceHash {
			_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, requests.RspUploadFileSliceData(&target), header.RspUploadFileSlice)
			// report upload result to SP

			_, newCtx := p2pserver.CreateNewContextPacketId(ctx)
			utils.DebugLog("ReqReportUploadSliceResultDataPP reqID =========", core.GetReqIdFromContext(newCtx))
			reportResultReq := requests.ReqReportUploadSliceResultDataPP(&target, totalCostTime)
			p2pserver.GetP2pServer(ctx).SendMessageToSPServer(newCtx, reportResultReq, header.ReqReportUploadSliceResult)
			metrics.StoredSliceCount.WithLabelValues("upload").Inc()
			instantInboundSpeed := float64(target.SliceSize) / math.Max(float64(totalCostTime), 1)
			metrics.InboundSpeed.WithLabelValues(reportResultReq.OpponentP2PAddress).Set(instantInboundSpeed)
			UpRecvCostTimeMap.mux.Lock()
			UpRecvCostTimeMap.dataMap.Delete(tkSlice)
			UpRecvCostTimeMap.mux.Unlock()
			utils.DebugLog("storage PP report to SP upload task finished: ", target.SliceInfo.SliceHash)
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
	metrics.UploadPerformanceLogNow(target.FileHash + ":RCV_RSP_SLICE:" + strconv.FormatInt(int64(target.SliceNumAddr.SliceNumber), 10))
	// verify node signature from sp
	if target.SpNodeSign == nil || target.PpNodeSign == nil {
		return
	}
	if err := verifyRspUploadSliceSign(&target); err != nil {
		utils.ErrorLog("RspUploadFileSlice", err.Error())
		return
	}

	pp.DebugLogf(ctx, "get RspUploadFileSlice for file %v  sliceNumber %v  size %v", target.FileHash, target.SliceNumAddr.SliceNumber, target.SliceSize)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		pp.ErrorLog(ctx, "RspUploadFileSlice failure:", target.Result.Msg)
		return
	}
	tkSlice := target.TaskId + strconv.FormatUint(target.SliceNumAddr.SliceNumber, 10)
	UpSendCostTimeMap.mux.Lock()
	defer UpSendCostTimeMap.mux.Unlock()
	if val, ok := UpSendCostTimeMap.dataMap.Load(tkSlice); ok {
		ctStat := val.(CostTimeStat)
		utils.DebugLogf("ctStat is %v", ctStat)
		if ctStat.PacketCount == 0 && ctStat.TotalCostTime > 0 {
			reportReq := requests.ReqReportUploadSliceResultData(&target, ctStat.TotalCostTime)
			p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, reportReq, header.ReqReportUploadSliceResult)
			instantOutboundSpeed := float64(target.SliceSize) / math.Max(float64(ctStat.TotalCostTime), 1)
			metrics.OutboundSpeed.WithLabelValues(reportReq.OpponentP2PAddress).Set(instantOutboundSpeed)

			UpSendCostTimeMap.dataMap.Delete(tkSlice)
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
		pp.DebugLog(ctx, "ResultState_RES_SUCCESS, sliceNumber，storageAddress，walletAddress", target.SliceNumAddr.SliceNumber, target.SliceNumAddr.PpInfo.NetworkAddress, target.SliceNumAddr.PpInfo.P2PAddress)
	} else {
		pp.Log(ctx, "ResultState_RES_FAIL : ", target.Result.Msg)
	}
}

func UploadFileSlice(ctx context.Context, tk *task.UploadSliceTask) error {
	tkDataLen := int(tk.SliceTotalSize)
	fileHash := tk.FileHash
	storageP2pAddress := tk.SliceNumAddr.PpInfo.P2PAddress
	storageNetworkAddress := tk.SliceNumAddr.PpInfo.NetworkAddress

	utils.DebugLog("reqID-"+tk.TaskID+" =========", strconv.FormatInt(core.GetReqIdFromContext(ctx), 10))
	tkSliceUID := tk.TaskID + strconv.FormatUint(tk.SliceNumAddr.SliceNumber, 10)
	tkSlice := TaskSlice{
		TkSliceUID:         tkSliceUID,
		IsUpload:           true,
		IsBackupOrTransfer: false,
	}
	var ctStat = CostTimeStat{}

	if tkDataLen <= setting.MAXDATA {
		tk.SliceOffsetInfo.SliceOffset.SliceOffsetStart = 0

		packetId, newCtx := p2pserver.CreateNewContextPacketId(ctx)
		PacketIdMap.Store(packetId, tkSlice)
		utils.DebugLogf("PacketIdMap.Store <==(%v, %v)", packetId, tkSlice)
		ctStat.PacketCount = ctStat.PacketCount + 1
		UpSendCostTimeMap.mux.Lock()
		UpSendCostTimeMap.dataMap.Store(tkSliceUID, ctStat)
		UpSendCostTimeMap.mux.Unlock()
		utils.DebugLogf("upSendPacketWgMap.Store <== K:%v, V:%v]", tkSliceUID, ctStat)
		return sendSlice(newCtx, requests.ReqUploadFileSliceData(tk, storageP2pAddress), fileHash, storageP2pAddress, storageNetworkAddress)
	}

	data, err := file.GetSliceDataFromTmp(tk.FileHash, tk.SliceOffsetInfo.SliceHash)
	if err != nil {
		return errors.Wrap(err, "failed get slice data from tmp")
	}
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
		packetId, newCtx := p2pserver.CreateNewContextPacketId(ctx)
		PacketIdMap.Store(packetId, tkSlice)
		utils.DebugLogf("PacketIdMap.Store <==(%v, %v)", packetId, tkSlice)
		UpSendCostTimeMap.mux.Lock()

		if val, ok := UpSendCostTimeMap.dataMap.Load(tk.TaskID + strconv.FormatUint(tk.SliceNumAddr.SliceNumber, 10)); ok {
			ctStat = val.(CostTimeStat)
		}
		ctStat.PacketCount = ctStat.PacketCount + 1
		UpSendCostTimeMap.dataMap.Store(tkSliceUID, ctStat)
		UpSendCostTimeMap.mux.Unlock()
		utils.DebugLogf("upSendPacketMap.Store <== K:%v, V:%v]", tkSliceUID, ctStat)
		if dataEnd < (tkDataLen + 1) {
			newTask.Data = data[dataStart:dataEnd]

			pp.DebugLogf(newCtx, "Uploading slice data %v-%v (total %v)", dataStart, dataEnd, newTask.SliceTotalSize)
			err := sendSlice(newCtx, requests.ReqUploadFileSliceData(newTask, storageP2pAddress), fileHash, storageP2pAddress, storageNetworkAddress)
			if err != nil {
				return err
			}
			dataStart += setting.MAXDATA
			dataEnd += setting.MAXDATA
		} else {
			pp.DebugLogf(newCtx, "Uploading slice data %v-%v (total %v)", dataStart, tkDataLen, newTask.SliceTotalSize)
			newTask.Data = data[dataStart:]
			return sendSlice(newCtx, requests.ReqUploadFileSliceData(newTask, storageP2pAddress), fileHash, storageP2pAddress, storageNetworkAddress)
		}
	}
}

func sendSlice(ctx context.Context, pb proto.Message, fileHash, p2pAddress, networkAddress string) error {
	pp.DebugLog(ctx, "sendSlice(pb proto.Message, fileHash, p2pAddress, networkAddress string)",
		fileHash, p2pAddress, networkAddress)
	key := "upload#" + fileHash + p2pAddress
	msg := pb.(*protos.ReqUploadFileSlice)
	metrics.UploadPerformanceLogNow(fileHash + ":SND_FILE_DATA:" + strconv.FormatInt(int64(msg.SliceInfo.SliceOffset.SliceOffsetStart+(msg.SliceNumAddr.SliceNumber-1)*33554432), 10) + ":" + networkAddress)
	return p2pserver.GetP2pServer(ctx).SendMessageByCachedConn(ctx, key, networkAddress, pb, header.ReqUploadFileSlice, HandleSendPacketCostTime)
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

	// verify pp address
	if !types.VerifyP2pAddrBytes(target.PpP2PPubkey, target.P2PAddress) {
		return errors.New("failed verifying pp's p2p address")
	}

	// verify node signature from the pp
	msg := utils.GetReqUploadFileSlicePpNodeSignMessage(target.P2PAddress, setting.P2PAddress, header.ReqUploadFileSlice)
	if !types.VerifyP2pSignBytes(target.PpP2PPubkey, target.PpNodeSign, msg) {
		return errors.New("failed verifying pp's node signature")
	}

	spP2pPubkey, err := requests.GetSpPubkey(target.SpP2PAddress)
	if err != nil {
		return errors.Wrap(err, "failed to get sp pubkey")
	}

	// verify sp address
	if !types.VerifyP2pAddrBytes(spP2pPubkey, target.SpP2PAddress) {
		return errors.New("failed verifying sp's p2p address")
	}

	// verify sp node signature
	msg = utils.GetReqUploadFileSliceSpNodeSignMessage(setting.P2PAddress, target.SpP2PAddress, target.FileHash, header.ReqUploadFileSlice)
	if !types.VerifyP2pSignBytes(spP2pPubkey, target.SliceNumAddr.SpNodeSign, msg) {
		return errors.New("failed verifying sp's node signature")
	}
	return nil
}

func verifyRspUploadSliceSign(target *protos.RspUploadFileSlice) error {

	// verify pp address
	if !types.VerifyP2pAddrBytes(target.PpP2PPubkey, target.P2PAddress) {
		return errors.New("failed verifying pp's p2p address")
	}

	// verify node signature from the pp
	msg := utils.GetRspUploadFileSliceNodeSignMessage(target.P2PAddress, setting.P2PAddress, header.RspUploadFileSlice)
	if !types.VerifyP2pSignBytes(target.PpP2PPubkey, target.PpNodeSign, msg) {
		return errors.New("failed verifying pp's node signature")
	}

	spP2pPubkey, err := requests.GetSpPubkey(target.SpP2PAddress)
	if err != nil {
		return errors.Wrap(err, "failed to get sp pubkey")
	}

	// verify sp address
	if !types.VerifyP2pAddrBytes(spP2pPubkey, target.SpP2PAddress) {
		return errors.New("failed verifying sp's p2p address")
	}

	// verify sp node signature
	msg = utils.GetReqUploadFileSliceSpNodeSignMessage(target.P2PAddress, target.SpP2PAddress, target.FileHash, header.ReqUploadFileSlice)
	if !types.VerifyP2pSignBytes(spP2pPubkey, target.SpNodeSign, msg) {
		return errors.New("failed verifying sp's node signature")
	}
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
