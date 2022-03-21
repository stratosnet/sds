package event

// Author j
import (
	"context"
	"crypto/ed25519"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
)

/* Commented out for backup logic redesign QB-897
// ReqTransferNotice  SP- original PP  OR  new PP - old PP
func ReqTransferNotice(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get ReqTransferNotice")
	var target protos.ReqTransferNotice
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	utils.DebugLog("target = ", target)
	if target.FromSp { // if msg from SP, then self is new storage PP
		utils.DebugLog("if msg from SP, then self is new storage PP")
		// response to SP first, check whether has capacity to store
		if target.StoragePpInfo.P2PAddress == setting.P2PAddress {

			utils.ErrorLog("target is myself, drop msg")
		}
		if task.CheckTransfer(&target) {
			// store this task
			task.TransferTaskMap[target.TransferCer] = &target
			rspTransferNotice(true, target.TransferCer, target.SpP2PAddress)
			// if accept task, send request transfer notice to original PP
			peers.TransferSendMessageToPPServ(target.StoragePpInfo.NetworkAddress, requests.ReqTransferNoticeData(&target))
			utils.DebugLog("rspTransferNotice sendTransferToPP ")
		} else {
			rspTransferNotice(false, target.TransferCer, target.SpP2PAddress)
		}
		return
	}
	// if msg from PP, then self is the original storage PP, transfer file to new storage PP
	utils.DebugLog("if msg from PP, then self is the original storage PP, transfer file to new storage PP")
	// check task with SP first
	peers.SendMessageToSPServer(requests.ReqValidateTransferCerData(&target), header.ReqValidateTransferCer)
	// store the task
	task.TransferTaskMap[target.TransferCer] = &target
	// store transfer target register peer wallet address
	peers.RegisterPeerMap.Store(target.StoragePpInfo.P2PAddress, core.NetIDFromContext(ctx))
}

// rspTransferNotice
func rspTransferNotice(agree bool, cer, spP2pAddress string) {
	peers.SendMessageToSPServer(requests.RspTransferNoticeData(agree, cer, spP2pAddress), header.RspTransferNotice)
}

// RspValidateTransferCer  SP-PP OR PP-PP
func RspValidateTransferCer(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspValidateTransferCer")
	var target protos.RspValidateTransferCer
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.ErrorLog("cert validation failed", target.Result.Msg)
		tTask, ok := task.TransferTaskMap[target.TransferCer]
		if !ok {
			return
		}
		//if cert from SP, then self is the original PP
		if tTask.FromSp {
			// task finished, clean
			delete(task.TransferTaskMap, target.TransferCer)
			return
		}
		// validation failed, clean task
		target.Result.State = protos.ResultState_RES_FAIL
		data, err := proto.Marshal(&target)
		if err != nil {
			return
		}
		msgBuf := msg.RelayMsgBuf{
			MSGHead: requests.PPMsgHeader(data, header.RspValidateTransferCer),
			MSGData: data,
		}
		peers.TransferSendMessageToClient(tTask.StoragePpInfo.P2PAddress, &msgBuf)
		delete(task.TransferTaskMap, target.TransferCer)

		return
	}

	utils.DebugLog("RspValidateTransferCer,TransferCer = ", target.TransferCer)
	tTask, ok := task.TransferTaskMap[target.TransferCer]
	if !ok {
		return
	}

	// if transfer cert from SP, then self is the transfer target
	if tTask.FromSp {
		utils.DebugLog("if transfer cert from SP, then self is the transfer target ")

		peers.SendMessage(conn, requests.ReqTransferDownloadData(target.TransferCer, target.SpP2PAddress), header.ReqTransferDownload)
	} else {
		// certificate validation success, resp to new PP to start download
		utils.DebugLog("cert validation success, resp to new PP to start download")
		peers.TransferSendMessageToClient(tTask.StoragePpInfo.P2PAddress, core.MessageFromContext(ctx))
	}
}

// ReqReportTransferResult
func ReqReportTransferResult(transferCer, spP2pAddress string, result bool, originDeleted bool) {
	tTask, ok := task.TransferTaskMap[transferCer]
	if !ok {
		return
	}
	req := &protos.ReqReportTransferResult{
		TransferCer:   transferCer,
		OriginDeleted: originDeleted,
		SpP2PAddress:  spP2pAddress,
	}
	if result {
		req.Result = &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		}
	} else {
		req.Result = &protos.Result{
			State: protos.ResultState_RES_FAIL,
		}
	}

	if tTask.FromSp {
		req.IsNew = true
		req.NewPp = &protos.PPBaseInfo{
			P2PAddress:     setting.P2PAddress,
			NetworkAddress: setting.NetworkAddress,
		}
	} else {
		req.IsNew = false
		req.NewPp = tTask.StoragePpInfo
	}
	peers.SendMessageToSPServer(req, header.ReqReportTransferResult)

	//todo: whether clean task after get report resp or not.  if not get report, whether add timeout mechanism
}

// RspReportTransferResult
func RspReportTransferResult(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspReportTransferResult")
	var target protos.RspReportTransferResult
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	// remove task
	delete(task.TransferTaskMap, target.TransferCer)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		utils.DebugLog("transfer successfully！！！", target.TransferCer)
	} else {
		utils.DebugLog("transfer failed！！！", target.TransferCer)
	}

}
*/

// ReqFileSliceBackupNotice An SP node wants this PP node to fetch the specified slice from the PP node who stores it
func ReqFileSliceBackupNotice(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get ReqFileSliceBackupNotice")
	var target protos.ReqFileSliceBackupNotice
	if !requests.UnmarshalData(ctx, &target) {
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

	if !task.CheckTransfer(&target) {
		utils.DebugLog("CheckTransfer failed")
		return
	}

	tTask := task.TransferTask{
		FromSp:           true,
		DeleteOrigin:     target.DeleteOrigin,
		PpInfo:           target.PpInfo,
		SliceStorageInfo: target.SliceStorageInfo,
		FileHash:         target.FileHash,
		SliceNum:         target.SliceNumber,
	}
	task.AddTransferTask(target.TaskId, target.SliceStorageInfo.SliceHash, tTask)

	peers.TransferSendMessageToPPServ(target.PpInfo.NetworkAddress, requests.ReqTransferDownloadData(&target, setting.P2PAddress))
}

// ReqTransferDownload
func ReqTransferDownload(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get ReqTransferDownload")
	var target protos.ReqTransferDownload
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	peers.UpdatePP(&types.PeerInfo{
		NetworkAddress: target.NewPp.NetworkAddress,
		P2pAddress:     target.NewPp.P2PAddress,
		RestAddress:    target.NewPp.RestAddress,
		WalletAddress:  target.NewPp.WalletAddress,
		NetId:          core.NetIDFromContext(ctx),
		Status:         types.PEER_CONNECTED,
	})
	tTask := task.TransferTask{
		FromSp:           false,
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
		if dataEnd >= (sliceDataLen + 1) {
			peers.SendMessage(conn, requests.RspTransferDownload(sliceData[dataStart:], target.TaskId, sliceHash,
				target.SpP2PAddress, uint64(dataStart), uint64(sliceDataLen)), header.RspTransferDownload)
			return
		}
		peers.SendMessage(conn, requests.RspTransferDownload(sliceData[dataStart:dataEnd], target.TaskId, sliceHash,
			target.SpP2PAddress, uint64(dataStart), uint64(sliceDataLen)), header.RspTransferDownload)
		dataStart += setting.MAXDATA
		dataEnd += setting.MAXDATA
	}
}

// RspTransferDownload
func RspTransferDownload(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspTransferDownload")
	var target protos.RspTransferDownload
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	if task.SaveTransferData(&target) {
		SendReportBackupSliceResult(target.TaskId, target.SliceHash, target.SpP2PAddress, true, false)
		peers.SendMessage(conn, requests.RspTransferDownloadResultData(target.TaskId, target.SliceHash, target.SpP2PAddress), header.RspTransferDownloadResult)
	}
}

// RspTransferDownloadResult original storage PP get this msg means download finished, can report and delete file
func RspTransferDownloadResult(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspTransferDownloadResult")
	var target protos.RspTransferDownloadResult
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	isSuccessful := target.Result.State == protos.ResultState_RES_SUCCESS
	if !isSuccessful {
		SendReportBackupSliceResult(target.TaskId, target.SliceHash, target.SpP2PAddress, isSuccessful, false)
		return
	}

	deleteOrigin := false
	if tTask, ok := task.GetTransferTask(target.TaskId, target.SliceHash); ok && tTask.DeleteOrigin {
		if err := file.DeleteSlice(tTask.SliceStorageInfo.SliceHash); err == nil {
			utils.Log("Delete original slice successfully")
			deleteOrigin = true
		} else {
			utils.ErrorLog("Fail to delete original slice ", err)
		}
	}
	SendReportBackupSliceResult(target.TaskId, target.SliceHash, target.SpP2PAddress, isSuccessful, deleteOrigin)
}

func SendReportBackupSliceResult(taskId, sliceHash, spP2pAddress string, result bool, originDeleted bool) {
	tTask, ok := task.GetTransferTask(taskId, sliceHash)
	if !ok {
		return
	}
	req := &protos.ReqReportBackupSliceResult{
		TaskId:        taskId,
		FileHash:      tTask.FileHash,
		SliceHash:     tTask.SliceStorageInfo.SliceHash,
		BackupSuccess: result,
		IsReceiver:    tTask.FromSp,
		OriginDeleted: originDeleted,
		SliceNumber:   tTask.SliceNum,
		SliceSize:     tTask.SliceStorageInfo.SliceSize,
		PpInfo:        &protos.PPBaseInfo{P2PAddress: setting.P2PAddress, WalletAddress: setting.WalletAddress},
		SpP2PAddress:  spP2pAddress,
	}

	peers.SendMessageToSPServer(req, header.ReqReportBackupSliceResult)
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
