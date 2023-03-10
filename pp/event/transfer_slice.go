package event

// Author j
import (
	"context"

	"github.com/stratosnet/sds/framework/client/cf"
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
	utilstypes "github.com/stratosnet/sds/utils/types"
	"github.com/tendermint/tendermint/types/time"
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

	// SPAM check
	if time.Now().Unix()-target.TimeStamp > setting.SPAM_THRESHOLD_SP_SIGN_LATENCY {
		utils.ErrorLog(ctx, "the slice backup request from sp was expired")
		return
	}

	if target.PpInfo.P2PAddress == setting.P2PAddress {
		utils.DebugLog("Ignoring slice backup notice because this node already owns the file")
		return
	}
	// get sp's p2p pubkey
	spP2pPubkey, err := requests.GetSpPubkey(target.SpP2PAddress)
	if err != nil {
		return
	}

	// verify sp address
	if !utilstypes.VerifyP2pAddrBytes(spP2pPubkey, target.SpP2PAddress) {
		return
	}

	// verify sp node signature
	nodeSign := target.NodeSign
	target.NodeSign = nil
	msg, err := utils.GetReqBackupSliceNoticeSpNodeSignMessage(target)
	if err != nil {
		utils.ErrorLog(ctx, "failed calculating signature from message")
		return
	}
	if !utilstypes.VerifyP2pSignBytes(spP2pPubkey, nodeSign, msg) {
		utils.ErrorLog(ctx, "failed verifying signature from sp")
		return
	}
	target.NodeSign = nodeSign

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
	}
	task.AddTransferTask(target.TaskId, target.SliceStorageInfo.SliceHash, tTask)

	//if the connection returns error, send a ReqTransferDownloadWrong message to sp to report the failure
	err = p2pserver.GetP2pServer(ctx).TransferSendMessageToPPServ(ctx, target.PpInfo.NetworkAddress, requests.ReqTransferDownloadData(target))
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
	setWriteHookForRspTransferSlice(conn)

	reqNotice := target.ReqFileSliceBackupNotice
	// SPAM check
	if time.Now().Unix()-reqNotice.TimeStamp > setting.SPAM_THRESHOLD_SP_SIGN_LATENCY {
		utils.ErrorLog(ctx, "the slice backup request from sp was expired")
		return
	}

	// get sp's p2p pubkey
	spP2pPubkey, err := requests.GetSpPubkey(reqNotice.SpP2PAddress)
	if err != nil {
		return
	}

	// verify sp address
	if !utilstypes.VerifyP2pAddrBytes(spP2pPubkey, reqNotice.SpP2PAddress) {
		return
	}

	// verify sp node signature
	nodeSign := reqNotice.NodeSign
	reqNotice.NodeSign = nil
	signmsg, err := utils.GetReqBackupSliceNoticeSpNodeSignMessage(reqNotice)
	if err != nil {
		utils.ErrorLog(ctx, "failed calculating signature from message")
		return
	}
	if !utilstypes.VerifyP2pSignBytes(spP2pPubkey, nodeSign, signmsg) {
		utils.ErrorLog(ctx, "failed verifying signature from sp")
		return
	}
	reqNotice.NodeSign = nodeSign

	// verify node sign between PPs
	if target.PpNodeSign == nil || target.P2PAddress == "" {
		utils.ErrorLog(ctx, "")
		return
	}

	if !utilstypes.VerifyP2pAddrBytes(target.PpP2PPubkey, target.P2PAddress) {
		utils.ErrorLogf("ppP2pPubkey validation failed, ppP2PAddress:[%v], ppP2PPubKey:[%v]", target.P2PAddress, target.PpP2PPubkey)
		return
	}

	msg := utils.GetReqTransferDownloadPpNodeSignMessage(target.P2PAddress, setting.P2PAddress, target.ReqFileSliceBackupNotice.SliceStorageInfo.SliceHash, header.ReqTransferDownload)
	if !utilstypes.VerifyP2pSignString(target.PpP2PPubkey, target.PpNodeSign, msg) {
		utils.ErrorLog("pp node signature validation failed, msg:", msg)
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
		IsReceiver:         false,
		DeleteOrigin:       reqNotice.DeleteOrigin,
		PpInfo:             reqNotice.PpInfo,
		SliceStorageInfo:   reqNotice.SliceStorageInfo,
		FileHash:           reqNotice.FileHash,
		SliceNum:           reqNotice.SliceNumber,
		ReceiverP2pAddress: target.NewPp.P2PAddress,
	}
	task.AddTransferTask(reqNotice.TaskId, reqNotice.SliceStorageInfo.SliceHash, tTask)

	sliceHash := reqNotice.SliceStorageInfo.SliceHash
	sliceData := task.GetTransferSliceData(reqNotice.TaskId, reqNotice.SliceStorageInfo.SliceHash)
	sliceDataLen := len(sliceData)
	utils.DebugLogf("sliceDataLen = %v  TaskId = %v", sliceDataLen, reqNotice.TaskId)

	tkSliceUID := reqNotice.TaskId + sliceHash
	dataStart := 0
	dataEnd := setting.MAXDATA
	for {
		packetId, newCtx := p2pserver.CreateNewContextPacketId(ctx)
		tkSlice := TaskSlice{
			TkSliceUID:         tkSliceUID,
			IsUpload:           false,
			IsBackupOrTransfer: true,
		}
		PacketIdMap.Store(packetId, tkSlice)
		utils.DebugLogf("PacketIdMap.Store <==(%v, %v)", packetId, tkSlice)
		costTimeStat := DownSendCostTimeMap.StartSendPacket(tkSliceUID)
		utils.DebugLogf("--- DownSendCostTimeMap.StartSendPacket--- taskId %v, sliceHash %v, costTimeStatAfter %v",
			reqNotice.TaskId, sliceHash, costTimeStat)
		if dataEnd > sliceDataLen {
			_ = p2pserver.GetP2pServer(ctx).SendMessage(newCtx, conn, requests.RspTransferDownload(sliceData[dataStart:], reqNotice.TaskId, sliceHash,
				reqNotice.SpP2PAddress, uint64(dataStart), uint64(sliceDataLen)), header.RspTransferDownload)
			return
		}
		_ = p2pserver.GetP2pServer(ctx).SendMessage(newCtx, conn, requests.RspTransferDownload(sliceData[dataStart:dataEnd], reqNotice.TaskId, sliceHash,
			reqNotice.SpP2PAddress, uint64(dataStart), uint64(sliceDataLen)), header.RspTransferDownload)
		dataStart += setting.MAXDATA
		dataEnd += setting.MAXDATA
	}
}

// RspTransferDownload The receiver PP gets this response from the uploader PP
func RspTransferDownload(ctx context.Context, conn core.WriteCloser) {
	costTime := core.GetRecvCostTimeFromContext(ctx)
	utils.Log("get RspTransferDownload")
	var target protos.RspTransferDownload
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	totalCostTIme := DownRecvCostTimeMap.AddCostTime(target.TaskId+target.SliceHash, costTime)

	// verify node sign between PPs
	if target.PpNodeSign == nil || target.P2PAddress == "" {
		utils.ErrorLog(ctx, "")
		return
	}

	if !utilstypes.VerifyP2pAddrBytes(target.PpP2PPubkey, target.P2PAddress) {
		utils.ErrorLogf("ppP2pPubkey validation failed, ppP2PAddress:[%v], ppP2PPubKey:[%v]", target.P2PAddress, target.PpP2PPubkey)
		return
	}

	msg := utils.GetRspTransferDownloadPpNodeSignMessage(target.P2PAddress, target.SpP2PAddress, target.SliceHash, header.ReqUploadFileSlice)
	if !utilstypes.VerifyP2pSignString(target.PpP2PPubkey, target.PpNodeSign, msg) {
		utils.ErrorLog("pp node signature validation failed, msg:", msg)
		return
	}

	err := task.SaveTransferData(&target)
	if err != nil {
		utils.ErrorLog("failed saving transfer data", err.Error())
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
	msg := utils.GetReqReportBackupSliceResultNodeSignMessage(setting.P2PAddress, spP2pAddress, sliceHash, header.ReqReportBackupSliceResult)
	req := &protos.ReqReportBackupSliceResult{
		TaskId:             taskId,
		FileHash:           tTask.FileHash,
		SliceHash:          tTask.SliceStorageInfo.SliceHash,
		BackupSuccess:      result,
		IsReceiver:         tTask.IsReceiver,
		OriginDeleted:      originDeleted,
		SliceNumber:        tTask.SliceNum,
		SliceSize:          tTask.SliceStorageInfo.SliceSize,
		PpInfo:             setting.GetPPInfo(),
		SpP2PAddress:       spP2pAddress,
		CostTime:           costTime,
		PpP2PAddress:       setting.P2PAddress,
		OpponentP2PAddress: opponentP2PAddress,
		P2PAddress:         setting.P2PAddress,
		PpP2PPubkey:        setting.P2PPublicKey,
		PpNodeSign:         utilstypes.BytesToP2pPrivKey(setting.P2PPrivateKey).Sign([]byte(msg)),
	}
	utils.DebugLogf("---SendReportBackupSliceResult, %v", req)
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

func handleBackupTransferSend(tkSlice TaskSlice, costTime int64) {
	DownSendCostTimeMap.FinishSendPacket(tkSlice.TkSliceUID, costTime)
}

func setWriteHookForRspTransferSlice(conn core.WriteCloser) {
	switch conn := conn.(type) {
	case *core.ServerConn:
		hookBackup := core.WriteHook{
			Message: header.RspTransferDownload,
			Fn:      HandleSendPacketCostTime,
		}
		var hooks []core.WriteHook
		hooks = append(hooks, hookBackup)
		conn.SetWriteHook(hooks)
	case *cf.ClientConn:
		hookBackup := cf.WriteHook{
			Message: header.RspTransferDownload,
			Fn:      HandleSendPacketCostTime,
		}
		var hooks []cf.WriteHook
		hooks = append(hooks, hookBackup)
		conn.SetWriteHook(hooks)
	}
}
