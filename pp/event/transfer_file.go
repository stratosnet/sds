package event

// Author j
import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/serv"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils"
)

// ReqTransferNotice  SP- original PP  OR  new PP - old PP
func ReqTransferNotice(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get ReqTransferNotice")
	var target protos.ReqTransferNotice
	if !unmarshalData(ctx, &target) {
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
			rspTransferNotice(true, target.TransferCer)
			// if accept task, send request transfer notice to original PP
			transferSendMessageToPPServ(target.StoragePpInfo.NetworkAddress, reqTransferNoticeData(&target))
			utils.DebugLog("rspTransferNotice sendTransferToPP ")
		} else {
			rspTransferNotice(false, target.TransferCer)
		}
		return
	}
	// if msg from PP, then self is the original storage PP, transfer file to new storage PP
	utils.DebugLog("if msg from PP, then self is the original storage PP, transfer file to new storage PP")
	// check task with SP first
	SendMessageToSPServer(reqValidateTransferCerData(&target), header.ReqValidateTransferCer)
	// store the task
	task.TransferTaskMap[target.TransferCer] = &target
	// store transfer target register peer wallet address
	serv.RegisterPeerMap.Store(target.StoragePpInfo.P2PAddress, core.NetIDFromContext(ctx))
}

// rspTransferNotice
func rspTransferNotice(agree bool, cer string) {
	SendMessageToSPServer(rspTransferNoticeData(agree, cer), header.RspTransferNotice)
}

// RspValidateTransferCer  SP-PP OR PP-PP
func RspValidateTransferCer(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspValidateTransferCer")
	var target protos.RspValidateTransferCer
	if !unmarshalData(ctx, &target) {
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
			MSGHead: PPMsgHeader(data, header.RspValidateTransferCer),
			MSGData: data,
		}
		transferSendMessageToClient(tTask.StoragePpInfo.P2PAddress, &msgBuf)
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

		sendMessage(conn, reqTransferDownloadData(target.TransferCer), header.ReqTransferDownload)
	} else {
		// certificate validation success, resp to new PP to start download
		utils.DebugLog("cert validation success, resp to new PP to start download")
		transferSendMessageToClient(tTask.StoragePpInfo.P2PAddress, core.MessageFromContext(ctx))
	}
}

// ReqReportTransferResult
func ReqReportTransferResult(transferCer string, result bool, originDeleted bool) {
	tTask, ok := task.TransferTaskMap[transferCer]
	if !ok {
		return
	}
	req := &protos.ReqReportTransferResult{
		TransferCer:   transferCer,
		OriginDeleted: originDeleted,
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
	SendMessageToSPServer(req, header.ReqReportTransferResult)

	//todo: whether clean task after get report resp or not.  if not get report, whether add timeout mechanism

}

// RspReportTransferResult
func RspReportTransferResult(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspReportTransferResult")
	var target protos.RspReportTransferResult
	if !unmarshalData(ctx, &target) {
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

// ReqTransferDownload
func ReqTransferDownload(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get ReqTransferDownload")
	var target protos.ReqTransferDownload
	if !unmarshalData(ctx, &target) {
		return
	}
	sliceData := task.GetTransferSliceData(target.TransferCer)
	sliceDataLen := len(sliceData)
	utils.DebugLog("————————————————————————————————————")
	utils.DebugLog("sliceDataLen == ", sliceDataLen)
	utils.DebugLog("TransferCer == ", target.TransferCer)
	dataStart := 0
	dataEnd := setting.MAXDATA
	for {
		if dataEnd >= (sliceDataLen + 1) {
			sendMessage(conn, rspTransferDownload(sliceData[dataStart:], target.TransferCer, uint64(dataStart), uint64(sliceDataLen)), header.RspTransferDownload)
			return
		}
		sendMessage(conn, rspTransferDownload(sliceData[dataStart:dataEnd], target.TransferCer, uint64(dataStart), uint64(sliceDataLen)), header.RspTransferDownload)
		dataStart += setting.MAXDATA
		dataEnd += setting.MAXDATA
	}
}

// RspTransferDownload
func RspTransferDownload(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspTransferDownload")
	var target protos.RspTransferDownload
	if !unmarshalData(ctx, &target) {
		return
	}
	if task.SaveTransferData(&target) {
		ReqReportTransferResult(target.TransferCer, true, false)
		sendMessage(conn, rspTransferDownloadResultData(target.TransferCer), header.RspTransferDownloadResult)
	}
}

// RspTransferDownloadResult original storage PP get this msg means download finished, can report and delete file
func RspTransferDownloadResult(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspTransferDownloadResult")
	var target protos.RspTransferDownloadResult
	if !unmarshalData(ctx, &target) {
		return
	}

	isSuccessful := target.Result.State == protos.ResultState_RES_SUCCESS
	if !isSuccessful {
		ReqReportTransferResult(target.TransferCer, isSuccessful, false)
		return
	}

	deleteOrigin := false
	if tTask, ok := task.TransferTaskMap[target.TransferCer]; ok && tTask.DeleteOrigin {
		//if msg from PP, then self is the original storage PP, delete original file if required
		if err := file.DeleteSlice(tTask.SliceStorageInfo.SliceHash); err == nil {
			utils.Log("Delete original slice successfully")
			deleteOrigin = true
		} else {
			utils.ErrorLog("Fail to delete original slice ", err)
		}
	}
	ReqReportTransferResult(target.TransferCer, isSuccessful, deleteOrigin)
}
