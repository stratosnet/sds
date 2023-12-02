package event

// Author j
import (
	"context"
	"strconv"
	"time"

	"github.com/alex023/clock"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/sds-msg/header"
	"github.com/stratosnet/sds/sds-msg/protos"
)

const CHECK_TRANFER_FAILURE_INTERVAL = 60 // in seconds

var (
	transferSliceSpamCheckMap  = utils.NewAutoCleanMap(setting.SpamThresholdSliceOperations)
	reportTransferFailureClock = clock.NewClock()
	reportTransferFailureJob   clock.Job
)

// NoticeFileSliceBackup An SP node wants this PP node to fetch the specified slice from the PP node who stores it.
// Both backups and transfers use the same method
func NoticeFileSliceBackup(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get NoticeFileSliceBackup")
	target := &protos.NoticeFileSliceBackup{}
	if err := VerifyMessage(ctx, header.NoticeFileSliceBackup, target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, target) {
		return
	}
	utils.DebugLog("target = ", target)

	if target.PpInfo.P2PAddress == p2pserver.GetP2pServer(ctx).GetP2PAddress().String() {
		utils.DebugLog("Ignoring slice backup notice because this node already owns the file")
		return
	}

	if !task.CheckTransfer(target) {
		utils.DebugLog("CheckTransfer failed")
		return
	}

	tTask := task.TransferTask{
		IsReceiver:         true,
		DeleteOrigin:       target.DeleteOrigin,
		PpInfo:             target.PpInfo,
		SliceStorageInfo:   target.SliceStorageInfo,
		FileHash:           target.FileHash,
		SliceNum:           target.SliceNumber,
		ReceiverP2pAddress: target.ToP2PAddress,
		SpP2pAddress:       target.SpP2PAddress,
		TaskId:             target.TaskId,
		AlreadySize:        uint64(0),
		LastTouchTime:      time.Now().Unix(),
	}
	task.AddTransferTask(target.TaskId, target.SliceStorageInfo.SliceHash, tTask)

	//if the connection returns error, send a ReqTransferDownloadWrong message to sp to report the failure
	err := p2pserver.GetP2pServer(ctx).TransferSendMessageToPPServ(ctx, target.PpInfo.NetworkAddress, requests.ReqTransferDownloadData(ctx, target))
	if err != nil {
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqTransferDownloadWrongData(ctx, target), header.ReqTransferDownloadWrong)
	}
}

// ReqTransferDownload Another PP wants to download a slice from the current PP
func ReqTransferDownload(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get ReqTransferDownload")
	var target protos.ReqTransferDownload
	if err := VerifyMessage(ctx, header.ReqTransferDownload, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	setWriteHookForRspTransferSlice(conn)

	noticeFileSliceBackup := target.NoticeFileSliceBackup

	// spam check
	key := noticeFileSliceBackup.TaskId + strconv.FormatInt(int64(noticeFileSliceBackup.SliceNumber), 10)
	if _, ok := transferSliceSpamCheckMap.Load(key); ok {
		rsp := &protos.RspTransferDownload{
			Result: &protos.Result{
				State: protos.ResultState_RES_FAIL,
				Msg:   "failed transferring file slice, re-transfer",
			},
		}
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, rsp, header.RspTransferDownload)
		return
	} else {
		var a any
		transferSliceSpamCheckMap.Store(key, a)
	}

	p2pserver.GetP2pServer(ctx).UpdatePP(ctx, &types.PeerInfo{
		NetworkAddress: target.NewPp.NetworkAddress,
		P2pAddress:     target.NewPp.P2PAddress,
		RestAddress:    target.NewPp.RestAddress,
		WalletAddress:  target.NewPp.WalletAddress,
		NetId:          core.NetIDFromContext(ctx),
		Status:         types.PEER_CONNECTED,
	})
	tTask := task.TransferTask{
		IsReceiver:         false,
		DeleteOrigin:       noticeFileSliceBackup.DeleteOrigin,
		PpInfo:             noticeFileSliceBackup.PpInfo,
		SliceStorageInfo:   noticeFileSliceBackup.SliceStorageInfo,
		FileHash:           noticeFileSliceBackup.FileHash,
		SliceNum:           noticeFileSliceBackup.SliceNumber,
		ReceiverP2pAddress: target.NewPp.P2PAddress,
		SpP2pAddress:       noticeFileSliceBackup.SpP2PAddress,
		AlreadySize:        uint64(0),
		LastTouchTime:      time.Now().Unix(),
	}
	task.AddTransferTask(noticeFileSliceBackup.TaskId, noticeFileSliceBackup.SliceStorageInfo.SliceHash, tTask)

	sliceHash := noticeFileSliceBackup.SliceStorageInfo.SliceHash
	sliceDataLen, buffer := task.GetTransferSliceData(noticeFileSliceBackup.TaskId, noticeFileSliceBackup.SliceStorageInfo.SliceHash)
	utils.DebugLogf("sliceDataLen = %v  TaskId = %v", sliceDataLen, noticeFileSliceBackup.TaskId)

	tkSliceUID := noticeFileSliceBackup.TaskId + sliceHash
	dataStart := 0
	dataEnd := setting.MaxData
	for _, data := range buffer {
		packetId, newCtx := p2pserver.CreateNewContextPacketId(ctx)
		tkSlice := TaskSlice{
			TkSliceUID:    tkSliceUID,
			SliceType:     SliceTransfer,
			TaskId:        noticeFileSliceBackup.TaskId,
			SliceHash:     noticeFileSliceBackup.SliceStorageInfo.SliceHash,
			SpP2pAddress:  noticeFileSliceBackup.SpP2PAddress,
			OriginDeleted: false,
		}
		PacketIdMap.Store(packetId, tkSlice)
		utils.DebugLogf("PacketIdMap.Store <==(%v, %v)", packetId, tkSlice)
		costTimeStat := DownSendCostTimeMap.StartSendPacket(tkSliceUID)
		utils.DebugLogf("--- DownSendCostTimeMap.StartSendPacket--- taskId %v, sliceHash %v, costTimeStatAfter %v",
			noticeFileSliceBackup.TaskId, sliceHash, costTimeStat)
		if int64(dataEnd) > sliceDataLen {
			_ = p2pserver.GetP2pServer(ctx).SendMessage(newCtx, conn, requests.RspTransferDownload(data, noticeFileSliceBackup.TaskId, sliceHash,
				noticeFileSliceBackup.SpP2PAddress, p2pserver.GetP2pServer(ctx).GetP2PAddress().String(), uint64(dataStart), uint64(sliceDataLen)), header.RspTransferDownload)
			return
		}
		_ = p2pserver.GetP2pServer(ctx).SendMessage(newCtx, conn, requests.RspTransferDownload(data, noticeFileSliceBackup.TaskId, sliceHash,
			noticeFileSliceBackup.SpP2PAddress, p2pserver.GetP2pServer(ctx).GetP2PAddress().String(), uint64(dataStart), uint64(sliceDataLen)), header.RspTransferDownload)
		dataStart += setting.MaxData
		dataEnd += setting.MaxData
		// add AlreadySize to transfer task
		task.AddAlreadySizeToTransferTask(noticeFileSliceBackup.TaskId, sliceHash, uint64(len(data)))
	}
}

// RspTransferDownload The receiver PP gets this response from the uploader PP
func RspTransferDownload(ctx context.Context, conn core.WriteCloser) {
	costTime := core.GetRecvCostTimeFromContext(ctx)
	utils.Log("get RspTransferDownload")
	var target protos.RspTransferDownload
	if err := VerifyMessage(ctx, header.RspTransferDownload, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	defer utils.ReleaseBuffer(target.Data)
	totalCostTIme := DownRecvCostTimeMap.AddCostTime(target.TaskId+target.SliceHash, costTime)

	completed, err := task.SaveTransferData(&target)
	if err != nil {
		utils.ErrorLog("saving transfer data", err.Error())
		return
	}
	if !completed {
		utils.DebugLogf("slice data saved, waiting for more data of this slice[%v]", target.SliceHash)
		return
	}
	// All data has been received
	SendReportBackupSliceResult(ctx, target.TaskId, target.SliceHash, target.SpP2PAddress, true, false, totalCostTIme)
	_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, requests.RspTransferDownloadResultData(target.TaskId, target.SliceHash, target.SpP2PAddress), header.RspTransferDownloadResult)
}

// RspTransferDownloadResult The receiver PP sends this msg when the download is finished. If successful, we can report the result and delete the file
func RspTransferDownloadResult(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspTransferDownloadResult")
	var target protos.RspTransferDownloadResult
	if err := VerifyMessage(ctx, header.RspTransferDownloadResult, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	tkSliceUID := target.TaskId + target.SliceHash
	totalCostTime, ok := DownSendCostTimeMap.GetCompletedTotalCostTime(tkSliceUID)
	if !ok {
		utils.DebugLog("slice not fully sent out")
		return
	}

	if target.Result.State != protos.ResultState_RES_SUCCESS {
		// Transfer failed
		SendReportBackupSliceResult(ctx, target.TaskId, target.SliceHash, target.SpP2PAddress, false, false, totalCostTime)
		return
	}

	deleteOrigin := false
	if tTask, ok := task.GetTransferTask(target.TaskId, target.SliceHash); ok && tTask.DeleteOrigin {
		if err := file.DeleteSlice(tTask.SliceStorageInfo.SliceHash); err == nil {
			utils.Log("Deleted original slice successfully")
			deleteOrigin = true
		} else {
			utils.ErrorLog("Failed to delete original slice ", err)
		}
	}
	SendReportBackupSliceResult(ctx, target.TaskId, target.SliceHash, target.SpP2PAddress, true, deleteOrigin, totalCostTime)
}

func SendReportBackupSliceResult(ctx context.Context, taskId, sliceHash, spP2pAddress string, result bool, originDeleted bool, costTime int64) {
	tTask, ok := task.GetTransferTask(taskId, sliceHash)
	if !ok {
		return
	}
	opponentP2PAddress := tTask.PpInfo.P2PAddress
	if !tTask.IsReceiver {
		opponentP2PAddress = tTask.ReceiverP2pAddress
	}
	req := &protos.ReqReportBackupSliceResult{
		TaskId:             taskId,
		FileHash:           tTask.FileHash,
		SliceHash:          tTask.SliceStorageInfo.SliceHash,
		BackupSuccess:      result,
		IsReceiver:         tTask.IsReceiver,
		OriginDeleted:      originDeleted,
		SliceNumber:        tTask.SliceNum,
		SliceSize:          tTask.SliceStorageInfo.SliceSize,
		PpInfo:             p2pserver.GetP2pServer(ctx).GetPPInfo(),
		SpP2PAddress:       spP2pAddress,
		CostTime:           costTime,
		PpP2PAddress:       p2pserver.GetP2pServer(ctx).GetP2PAddress().String(),
		OpponentP2PAddress: opponentP2PAddress,
		P2PAddress:         p2pserver.GetP2pServer(ctx).GetP2PAddress().String(),
	}
	utils.DebugLogf("---SendReportBackupSliceResult, %v", req)
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, req, header.ReqReportBackupSliceResult)
}

// RspReportBackupSliceResult
func RspReportBackupSliceResult(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspReportBackupSliceResult")
	var target protos.RspReportBackupSliceResult
	if err := VerifyMessage(ctx, header.RspReportBackupSliceResult, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	// remove task
	task.CleanTransferTask(target.TaskId, target.SliceHash)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		utils.DebugLog("transfer successful!", target.TaskId)
	} else {
		utils.DebugLog("transfer failed!", target.TaskId)
	}
}

func handleBackupTransferSend(tkSlice TaskSlice, costTime int64) {
	DownSendCostTimeMap.FinishSendPacket(tkSlice.TkSliceUID, costTime)
}

func setWriteHookForRspTransferSlice(conn core.WriteCloser) {
	switch conn := conn.(type) {
	case *core.ServerConn:
		hookBackup := core.WriteHook{
			MessageId: header.RspTransferDownload.Id,
			Fn:        HandleSendPacketCostTime,
		}
		var hooks []core.WriteHook
		hooks = append(hooks, hookBackup)
		conn.SetWriteHook(hooks)
	case *cf.ClientConn:
		hookBackup := cf.WriteHook{
			MessageId: header.RspTransferDownload.Id,
			Fn:        HandleSendPacketCostTime,
		}
		var hooks []cf.WriteHook
		hooks = append(hooks, hookBackup)
		conn.SetWriteHook(hooks)
	}
}

func StartReportTransferFailureJob(ctx context.Context) {
	utils.Log("Starting ReportTransferFailureJob......")
	reportTransferFailureJob, _ = reportTransferFailureClock.AddJobRepeat(CHECK_TRANFER_FAILURE_INTERVAL*time.Second, 0, CheckTransferTimeout(ctx))
}

func StopReportTransferFailureJob() {
	if reportTransferFailureJob != nil {
		utils.Log("Stopping ReportTransferFailureJob......")
		reportTransferFailureJob.Cancel()
	}
}

func CheckTransferTimeout(ctx context.Context) func() {
	return func() {
		timeoutTaskUIDs := task.GetTimeoutTransfer()
		sendTransferFailureReportToSP(ctx, timeoutTaskUIDs)
	}
}

func sendTransferFailureReportToSP(ctx context.Context, taskSliceUIDs []string) {
	for _, taskSliceUID := range taskSliceUIDs {
		tTask, ok := task.GetTransferTaskByTaskSliceUID(taskSliceUID)
		if !ok {
			continue
		}
		utils.DebugLogf("--- reporting backup failure for task[%v]-sliceHash[%v] to sp[%v], isReceiver=%v",
			tTask.TaskId, tTask.SliceStorageInfo.SliceHash, tTask.SpP2pAddress, tTask.IsReceiver)
		SendReportBackupSliceResult(ctx, tTask.TaskId, tTask.SliceStorageInfo.SliceHash, tTask.SpP2pAddress, false, false, 0)
		// delete KV from maps
		task.CleanTransferTaskByTaskSliceUID(taskSliceUID)
	}
}
