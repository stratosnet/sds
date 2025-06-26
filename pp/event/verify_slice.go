package event

import (
	"context"
	"time"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/msg/header"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/sds-msg/protos"
)

func NoticeFileSliceVerify(ctx context.Context, conn core.WriteCloser) {
	target := &protos.NoticeFileSliceVerify{}
	if err := VerifyMessage(ctx, header.NoticeFileSliceVerify, target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, target) {
		return
	}

	if target.PpInfo.P2PAddress == p2pserver.GetP2pServer(ctx).GetP2PAddress().String() {
		utils.DebugLog("Ignoring verify notice because this node already owns the file")
		return
	}

	tTask := task.VerifyTask{
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
	task.AddVerifyTask(target.TaskId, target.SliceStorageInfo.SliceHash, tTask)
	p2pserver.GetP2pServer(ctx).SendMessageToPPServ(ctx, target.PpInfo.NetworkAddress, requests.ReqVerifyDownloadData(ctx, target), nil, nil, header.MsgType{Id: 0, Name: ""})
}

func ReqVerifyDownload(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqVerifyDownload
	if err := VerifyMessage(ctx, header.ReqVerifyDownload, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	setWriteHookForRspTransferSlice(conn)
	noticeVerify := target.NoticeFileSliceVerify

	tTask := task.VerifyTask{
		IsReceiver:         false,
		FileHash:           noticeVerify.FileHash,
		DeleteOrigin:       noticeVerify.DeleteOrigin,
		PpInfo:             noticeVerify.PpInfo,
		SliceStorageInfo:   noticeVerify.SliceStorageInfo,
		SliceNum:           noticeVerify.SliceNumber,
		ReceiverP2pAddress: target.NewPp.P2PAddress,
		SpP2pAddress:       noticeVerify.SpP2PAddress,
		AlreadySize:        uint64(0),
	}
	task.AddVerifyTask(noticeVerify.TaskId, noticeVerify.SliceStorageInfo.SliceHash, tTask)
	sliceHash := noticeVerify.SliceStorageInfo.SliceHash
	sliceDataLen, buffer := task.GetVerifySliceData(noticeVerify.TaskId, noticeVerify.SliceStorageInfo.SliceHash)
	tkSliceUID := noticeVerify.TaskId + sliceHash
	dataStart := 0
	dataEnd := setting.MaxData
	for _, data := range buffer {
		packetId, newCtx := p2pserver.CreateNewContextPacketId(ctx)
		tkSlice := TaskSlice{
			TkSliceUID:    tkSliceUID,
			SliceType:     SliceTransfer,
			TaskId:        noticeVerify.TaskId,
			SliceHash:     noticeVerify.SliceStorageInfo.SliceHash,
			SpP2pAddress:  noticeVerify.SpP2PAddress,
			OriginDeleted: false,
		}
		PacketIdMap.Store(packetId, tkSlice)
		if int64(dataEnd) > sliceDataLen {
			_ = p2pserver.GetP2pServer(ctx).SendMessage(
				newCtx,
				conn,
				requests.RspVerifyDownload(data, noticeVerify.TaskId, sliceHash, noticeVerify.SpP2PAddress, p2pserver.GetP2pServer(ctx).GetP2PAddress().String(), uint64(dataStart), uint64(sliceDataLen), noticeVerify.SliceNumber),
				header.RspVerifyDownload,
			)
			return
		}
		_ = p2pserver.GetP2pServer(ctx).SendMessage(
			newCtx,
			conn,
			requests.RspVerifyDownload(data, noticeVerify.TaskId, sliceHash, noticeVerify.SpP2PAddress, p2pserver.GetP2pServer(ctx).GetP2PAddress().String(), uint64(dataStart), uint64(sliceDataLen), noticeVerify.SliceNumber),
			header.RspVerifyDownload,
		)
		dataStart += setting.MaxData
		dataEnd += setting.MaxData
		// add AlreadySize to transfer task
		task.AddAlreadySizeToTransferTask(noticeVerify.TaskId, sliceHash, uint64(len(data)))
	}
}

func RspVerifyDownload(ctx context.Context, conn core.WriteCloser) {
	//costTime := core.GetRecvCostTimeFromContext(ctx)
	var target protos.RspVerifyDownload
	if err := VerifyMessage(ctx, header.RspVerifyDownload, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	if target.Result != nil && target.Result.State == protos.ResultState_RES_FAIL {
		utils.ErrorLog("received failed transfer download,", target.Result.Msg)
		return
	}
	if target.Data == nil {
		utils.ErrorLog("no data contained in the message")
		return
	}
	defer utils.ReleaseBuffer(target.Data)

	completed, err := task.SaveVerifyData(&target)
	if err != nil {
		utils.ErrorLog("saving transfer data", err.Error())
		return
	}
	if !completed {
		//utils.DebugLogf("slice data saved, waiting for more data of this slice[%v]", target.SliceHash)
		return
	}

	// data is received
	SendReportVerifyResult(ctx, target.TaskId, target.SliceHash, target.SpP2PAddress, true, target.SliceSize)

	_ = p2pserver.GetP2pServer(ctx).SendMessage(ctx, conn, requests.RspVerifyDownloadResultData(target.TaskId, target.SliceHash, target.SpP2PAddress), header.RspVerifyDownloadResult)
}

func SendReportVerifyResult(ctx context.Context, taskId, sliceHash, spP2pAddress string, result bool, sliceSize uint64) {
	tTask, ok := task.GetVerifyTask(taskId, sliceHash)
	if !ok {
		utils.ErrorLog("Transfer/backup task is already removed.")
		return
	}
	opponentP2PAddress := tTask.PpInfo.P2PAddress
	if !tTask.IsReceiver {
		opponentP2PAddress = tTask.ReceiverP2pAddress
	}
	req := &protos.ReqReportVerifyResult{
		TaskId:             taskId,
		FileHash:           tTask.FileHash,
		SliceHash:          tTask.SliceStorageInfo.SliceHash,
		BackupSuccess:      result,
		IsReceiver:         tTask.IsReceiver,
		SliceNumber:        tTask.SliceNum,
		SliceSize:          sliceSize,
		PpInfo:             p2pserver.GetP2pServer(ctx).GetPPInfo(),
		SpP2PAddress:       spP2pAddress,
		PpP2PAddress:       p2pserver.GetP2pServer(ctx).GetP2PAddress().String(),
		OpponentP2PAddress: opponentP2PAddress,
		P2PAddress:         p2pserver.GetP2pServer(ctx).GetP2PAddress().String(),
	}
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, req, header.ReqReportVerifyResult)
}

func RspVerifyDownloadResult(ctx context.Context, conn core.WriteCloser) {
	utils.DetailLog("get RspTransferDownloadResult")
	var target protos.RspVerifyDownloadResult
	if err := VerifyMessage(ctx, header.RspTransferDownloadResult, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
}
