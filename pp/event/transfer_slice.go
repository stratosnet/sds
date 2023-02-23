package event

// Author j
import (
	"context"
	"crypto/ed25519"

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
)

// ReqFileSliceBackupNotice An SP node wants this PP node to fetch the specified slice from the PP node who stores it.
// Both backups and transfers use the same method
func ReqFileSliceBackupNotice(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get ReqFileSliceBackupNotice")
	target := &protos.ReqFileSliceBackupNotice{}
	if !requests.UnmarshalData(ctx, target) {
		return
	}
	utils.DebugLog("target = ", target)

	if target.PpInfo.P2PAddress == setting.P2PAddress {
		utils.DebugLog("Ignoring slice backup notice because this node already owns the file")
		return
	}

	signMessage := target.FileHash + "#" + target.SliceStorageInfo.SliceHash + "#" + target.SpP2PAddress
	if !ed25519.Verify(target.Pubkey, []byte(signMessage), target.Sign) {
		utils.ErrorLog("Invalid slice backup notice signature")
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
	if !requests.UnmarshalData(ctx, &target) {
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
		DeleteOrigin:     target.DeleteOrigin,
		PpInfo:           target.OriginalPp,
		SliceStorageInfo: target.SliceStorageInfo,
		FileHash:         target.FileHash,
		SliceNum:         target.SliceNum,
	}
	task.AddTransferTask(target.TaskId, target.SliceStorageInfo.SliceHash, tTask)

	sliceHash := target.SliceStorageInfo.SliceHash
	sliceData := task.GetTransferSliceData(target.TaskId, target.SliceStorageInfo.SliceHash)
	sliceDataLen := len(sliceData)
	utils.DebugLogf("sliceDataLen = %v  TaskId = %v", sliceDataLen, target.TaskId)

	dataStart := 0
	dataEnd := setting.MAXDATA
	for {
		if dataEnd > sliceDataLen {
			_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, requests.RspTransferDownload(sliceData[dataStart:], target.TaskId, sliceHash,
				target.SpP2PAddress, uint64(dataStart), uint64(sliceDataLen)), header.RspTransferDownload)
			return
		}
		_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, requests.RspTransferDownload(sliceData[dataStart:dataEnd], target.TaskId, sliceHash,
			target.SpP2PAddress, uint64(dataStart), uint64(sliceDataLen)), header.RspTransferDownload)
		dataStart += setting.MAXDATA
		dataEnd += setting.MAXDATA
	}
}

// RspTransferDownload The receiver PP gets this response from the uploader PP
func RspTransferDownload(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspTransferDownload")
	var target protos.RspTransferDownload
	if !requests.UnmarshalData(ctx, &target) {
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
	}

	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, req, header.ReqReportBackupSliceResult)
}

// RspReportBackupSliceResult
func RspReportBackupSliceResult(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspReportBackupSliceResult")
	var target protos.RspReportBackupSliceResult
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
