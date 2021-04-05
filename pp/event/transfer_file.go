package event

// Author j
import (
	"context"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg"
	"github.com/qsnetwork/sds/msg/header"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/pp/serv"
	"github.com/qsnetwork/sds/pp/setting"
	"github.com/qsnetwork/sds/pp/task"
	"github.com/qsnetwork/sds/utils"

	"github.com/golang/protobuf/proto"
)

// ReqTransferNotice  SP- original PP  OR  new PP - old PP
func ReqTransferNotice(ctx context.Context, conn spbf.WriteCloser) {
	utils.DebugLog("get ReqTransferNotice")
	var target protos.ReqTransferNotice
	if unmarshalData(ctx, &target) {
		utils.DebugLog("target = ", target)
		if target.FromSp { // if msg from SP, then self is new storage PP
			utils.DebugLog("if msg from SP, then self is new storage PP")
			// response to SP first, check whether has capacity to store
			if target.StoragePpInfo.WalletAddress == setting.WalletAddress {

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
		} else { // if msg from PP, then self is the original storage PP, transfer file to new storage PP
			utils.DebugLog("if msg from PP, then self is the original storage PP, transfer file to new storage PP")
			// check task with SP first
			SendMessageToSPServer(reqValidateTransferCerData(&target), header.ReqValidateTransferCer)
			// store the task
			task.TransferTaskMap[target.TransferCer] = &target
			// store transfer target register peer wallet address
			serv.RegisterPeerMap.Store(target.StoragePpInfo.WalletAddress, spbf.NetIDFromContext(ctx))
		}
	}
}

// rspTransferNotice
func rspTransferNotice(agree bool, cer string) {
	SendMessageToSPServer(rspTransferNoticeData(agree, cer), header.RspTransferNotice)
}

// RspValidateTransferCer  SP-PP OR PP-PP
func RspValidateTransferCer(ctx context.Context, conn spbf.WriteCloser) {
	utils.Log("get RspValidateTransferCer")
	var target protos.RspValidateTransferCer
	if unmarshalData(ctx, &target) {
		if target.Result.State == protos.ResultState_RES_SUCCESS {

			utils.DebugLog("RspValidateTransferCer,TransferCer = ", target.TransferCer)
			if tTask, ok := task.TransferTaskMap[target.TransferCer]; ok {
				// if transfer cert from SP, then self is the transfer target
				if tTask.FromSp {
					utils.DebugLog("if transfer cert from SP, then self is the transfer target ")

					sendMessage(conn, reqTransferDownloadData(target.TransferCer), header.ReqTransferDownload)
				} else {
					// certificate validation success, resp to new PP to start download
					utils.DebugLog("cert validation success, resp to new PP to start download")
					transferSendMessageToClient(tTask.StoragePpInfo.WalletAddress, spbf.MessageFromContext(ctx))
				}
			}
		} else {
			utils.ErrorLog("cert validation failed", target.Result.Msg)
			if tTask, ok := task.TransferTaskMap[target.TransferCer]; ok {
				// if cert from SP, then self is the original PP
				if tTask.FromSp {
					// task finished, clean
					delete(task.TransferTaskMap, target.TransferCer)
				} else {
					// validation failed, clean task
					target.Result.State = protos.ResultState_RES_FAIL
					data, err := proto.Marshal(&target)
					if utils.CheckError(err) {
						return
					}
					msgBuf := msg.RelayMsgBuf{
						MSGHead: PPMsgHeader(data, header.RspValidateTransferCer),
						MSGData: data,
					}
					transferSendMessageToClient(tTask.StoragePpInfo.WalletAddress, &msgBuf)
					delete(task.TransferTaskMap, target.TransferCer)
				}
			}
		}
	}
}

// ReqReportTransferResult
func ReqReportTransferResult(transferCer string, result bool) {
	if tTask, ok := task.TransferTaskMap[transferCer]; ok {
		req := &protos.ReqReportTransferResult{
			TransferCer: transferCer,
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
				WalletAddress:  setting.WalletAddress,
				NetworkAddress: setting.NetworkAddress,
			}
		} else {
			req.IsNew = false
			req.NewPp = tTask.StoragePpInfo
		}
		SendMessageToSPServer(req, header.ReqReportTransferResult)

		//todo: whether clean task after get report resp or not.  if not get report, whether add timeout mechanism
	}
}

// RspReportTransferResult
func RspReportTransferResult(ctx context.Context, conn spbf.WriteCloser) {
	utils.Log("get RspReportTransferResult")
	var target protos.RspReportTransferResult
	if unmarshalData(ctx, &target) {
		// 移除任务
		delete(task.TransferTaskMap, target.TransferCer)
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			utils.DebugLog("transfer successfully！！！", target.TransferCer)
		} else {
			utils.DebugLog("transfer failed！！！", target.TransferCer)
		}
	}
}

// ReqTransferDownload
func ReqTransferDownload(ctx context.Context, conn spbf.WriteCloser) {
	utils.Log("get ReqTransferDownload")
	var target protos.ReqTransferDownload
	if unmarshalData(ctx, &target) {
		sliceData := task.GetTransferSliceData(target.TransferCer)
		sliceDataLen := len(sliceData)
		utils.DebugLog("————————————————————————————————————")
		utils.DebugLog("sliceDataLen == ", sliceDataLen)
		utils.DebugLog("TransferCer == ", target.TransferCer)
		dataStart := 0
		dataEnd := setting.MAXDATA
		for {
			if dataEnd < (sliceDataLen + 1) {
				sendMessage(conn, rspTransferDownload(sliceData[dataStart:dataEnd], target.TransferCer, uint64(dataStart), uint64(sliceDataLen)), header.RspTransferDownload)
				dataStart += setting.MAXDATA
				dataEnd += setting.MAXDATA
			} else {
				sendMessage(conn, rspTransferDownload(sliceData[dataStart:], target.TransferCer, uint64(dataStart), uint64(sliceDataLen)), header.RspTransferDownload)
				return
			}
		}
	}
}

// RspTransferDownload
func RspTransferDownload(ctx context.Context, conn spbf.WriteCloser) {
	utils.Log("get RspTransferDownload")
	var target protos.RspTransferDownload
	if unmarshalData(ctx, &target) {
		if task.SaveTransferData(&target) {
			ReqReportTransferResult(target.TransferCer, true)
			sendMessage(conn, rspTransferDownloadResultData(target.TransferCer), header.RspTransferDownloadResult)
		}
	}
}

// RspTransferDownloadResult original storage PP get this msg means download finished, can report and delete file
func RspTransferDownloadResult(ctx context.Context, conn spbf.WriteCloser) {
	utils.Log("get RspTransferDownloadResult")
	var target protos.RspTransferDownloadResult
	if unmarshalData(ctx, &target) {
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			ReqReportTransferResult(target.TransferCer, true)
		} else {
			ReqReportTransferResult(target.TransferCer, false)
		}
	}
}
