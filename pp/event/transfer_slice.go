package event

// Author j
import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
	"github.com/tendermint/tendermint/types/time"
)

// ReqFileSliceBackupNotice An SP node wants this PP node to fetch the specified slice from the PP node who stores it.
// Both backups and transfers use the same method
func ReqFileSliceBackupNotice(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get ReqFileSliceBackupNotice")
	target := &protos.ReqFileSliceBackupNotice{}
	if err := VerifyMessage(ctx, header.ReqFileSliceBackupNotice, target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
	}
	if !requests.UnmarshalData(ctx, target) {
		return
	}
	utils.DebugLog("target = ", target)

	// SPAM check
	if time.Now().Unix()-target.TimeStamp > setting.SPAM_THRESHOLD_SP_SIGN_LATENCY {
		utils.ErrorLog(ctx, "the slice backup request from sp was expired")
		return
	}

	if target.PpInfo.P2PAddress == setting.P2PAddress {
		utils.DebugLog("Ignoring slice backup notice because this node already owns the file")
		return
	}

	if !task.CheckTransfer(target) {
		utils.DebugLog("CheckTransfer failed")
		return
	}

	tTask := task.TransferTask{
		IsReceiver:       true,
		DeleteOrigin:     target.DeleteOrigin,
		PpInfo:           target.PpInfo,
		SliceStorageInfo: target.SliceStorageInfo,
		FileHash:         target.FileHash,
		SliceNum:         target.SliceNumber,
	}
	task.AddTransferTask(target.TaskId, target.SliceStorageInfo.SliceHash, tTask)

	//if the connection returns error, send a ReqTransferDownloadWrong message to sp to report the failure
	err := p2pserver.GetP2pServer(ctx).TransferSendMessageToPPServ(ctx, target.PpInfo.NetworkAddress, requests.ReqTransferDownloadData(target))
	if err != nil {
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqTransferDownloadWrongData(target), header.ReqTransferDownloadWrong)
	}
}

// ReqTransferDownload Another PP wants to download a slice from the current PP
func ReqTransferDownload(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get ReqTransferDownload")
	var target protos.ReqTransferDownload
	if err := VerifyMessage(ctx, header.ReqTransferDownload, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	reqNotice := target.ReqFileSliceBackupNotice
	// SPAM check
	if time.Now().Unix()-reqNotice.TimeStamp > setting.SPAM_THRESHOLD_SP_SIGN_LATENCY {
		utils.ErrorLog(ctx, "the slice backup request from sp was expired")
		return
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
		IsReceiver:       false,
		DeleteOrigin:     reqNotice.DeleteOrigin,
		PpInfo:           reqNotice.PpInfo,
		SliceStorageInfo: reqNotice.SliceStorageInfo,
		FileHash:         reqNotice.FileHash,
		SliceNum:         reqNotice.SliceNumber,
	}
	task.AddTransferTask(reqNotice.TaskId, reqNotice.SliceStorageInfo.SliceHash, tTask)

	sliceHash := reqNotice.SliceStorageInfo.SliceHash
	sliceData := task.GetTransferSliceData(reqNotice.TaskId, reqNotice.SliceStorageInfo.SliceHash)
	sliceDataLen := len(sliceData)
	utils.DebugLogf("sliceDataLen = %v  TaskId = %v", sliceDataLen, reqNotice.TaskId)

	dataStart := 0
	dataEnd := setting.MAXDATA
	for {
		if dataEnd > sliceDataLen {
			_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, requests.RspTransferDownload(sliceData[dataStart:], reqNotice.TaskId, sliceHash,
				reqNotice.SpP2PAddress, uint64(dataStart), uint64(sliceDataLen)), header.RspTransferDownload)
			return
		}
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, requests.RspTransferDownload(sliceData[dataStart:dataEnd], reqNotice.TaskId, sliceHash,
			reqNotice.SpP2PAddress, uint64(dataStart), uint64(sliceDataLen)), header.RspTransferDownload)
		dataStart += setting.MAXDATA
		dataEnd += setting.MAXDATA
	}
}

// RspTransferDownload The receiver PP gets this response from the uploader PP
func RspTransferDownload(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspTransferDownload")
	var target protos.RspTransferDownload
	if err := VerifyMessage(ctx, header.RspTransferDownload, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	// verify node sign between PPs
	if target.P2PAddress == "" {
		utils.ErrorLog(ctx, "")
		return
	}

	err := task.SaveTransferData(&target)
	if err != nil {
		utils.ErrorLog("failed saving transfer data", err.Error())
		return
	}
	// All data has been received
	SendReportBackupSliceResult(ctx, target.TaskId, target.SliceHash, target.SpP2PAddress, true, false)
	_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, requests.RspTransferDownloadResultData(target.TaskId, target.SliceHash, target.SpP2PAddress), header.RspTransferDownloadResult)
}

// RspTransferDownloadResult The receiver PP sends this msg when the download is finished. If successful, we can report the result and delete the file
func RspTransferDownloadResult(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspTransferDownloadResult")
	var target protos.RspTransferDownloadResult
	if err := VerifyMessage(ctx, header.RspTransferDownloadResult, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	if target.Result.State != protos.ResultState_RES_SUCCESS {
		// Transfer failed
		SendReportBackupSliceResult(ctx, target.TaskId, target.SliceHash, target.SpP2PAddress, false, false)
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
	SendReportBackupSliceResult(ctx, target.TaskId, target.SliceHash, target.SpP2PAddress, true, deleteOrigin)
}

func SendReportBackupSliceResult(ctx context.Context, taskId, sliceHash, spP2pAddress string, result bool, originDeleted bool) {
	tTask, ok := task.GetTransferTask(taskId, sliceHash)
	if !ok {
		return
	}
	req := &protos.ReqReportBackupSliceResult{
		TaskId:        taskId,
		FileHash:      tTask.FileHash,
		SliceHash:     tTask.SliceStorageInfo.SliceHash,
		BackupSuccess: result,
		IsReceiver:    tTask.IsReceiver,
		OriginDeleted: originDeleted,
		SliceNumber:   tTask.SliceNum,
		SliceSize:     tTask.SliceStorageInfo.SliceSize,
		PpInfo:        setting.GetPPInfo(),
		SpP2PAddress:  spP2pAddress,
		P2PAddress:    setting.P2PAddress,
	}

	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, req, header.ReqReportBackupSliceResult)
}

// RspReportBackupSliceResult
func RspReportBackupSliceResult(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspReportBackupSliceResult")
	var target protos.RspReportBackupSliceResult
	if err := VerifyMessage(ctx, header.RspReportBackupSliceResult, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
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
