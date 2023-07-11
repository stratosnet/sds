package event

// Author j
import (
	"context"
	"encoding/json"
	"net/http"
	"time"

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
	"github.com/stratosnet/sds/utils/datamesh"
	"github.com/stratosnet/sds/utils/httpserv"
	"github.com/stratosnet/sds/utils/types"
)

// GetFileStorageInfo p to pp. The downloader is assumed the default wallet of this node, if this function is invoked.
func GetFileStorageInfo(ctx context.Context, path, savePath, saveAs string, w http.ResponseWriter) {
	utils.DebugLog("GetFileStorageInfo")

	if !setting.CheckLogin() {
		if w != nil {
			_, _ = w.Write(httpserv.NewJson(nil, setting.FAILCode, "login first").ToBytes())
		}
		return
	}
	if len(path) < setting.DownloadPathMinLen {
		utils.DebugLog("invalid path length")
		return
	}
	_, walletAddress, fileHash, _, err := datamesh.ParseFileHandle(path)
	if err != nil {
		pp.ErrorLog(ctx, "please input correct download link, eg: sdm://address/fileHash|filename(optional)")
		if w != nil {
			_, _ = w.Write(httpserv.NewJson(nil, setting.FAILCode, "please input correct download link, eg:  sdm://address/fileHash|filename(optional)").ToBytes())
		}
		return
	}
	metrics.DownloadPerformanceLogNow(fileHash + ":RCV_CMD_START:")

	pp.DebugLog(ctx, "path:", path)

	if ok := task.CheckDownloadTask(fileHash, walletAddress, task.LOCAL_REQID); ok {
		msg := "The previous download task hasn't finished, please check back later"
		pp.ErrorLog(ctx, msg)
		if w != nil {
			_, _ = w.Write(httpserv.NewJson(nil, setting.FAILCode, msg).ToBytes())
		}
		return
	}

	nowSec := time.Now().Unix()
	// sign the wallet signature by wallet private key
	wsignMsg := utils.GetFileDownloadWalletSignMessage(fileHash, setting.WalletAddress, "", nowSec) // need to retrieve sn first
	wsign, err := types.BytesToAccPriveKey(setting.WalletPrivateKey).Sign([]byte(wsignMsg))
	if err != nil {
		return
	}
	req := requests.ReqFileStorageInfoData(ctx, path, savePath, saveAs, setting.WalletAddress, setting.WalletPublicKey, wsign, nil, nowSec)
	metrics.DownloadPerformanceLogNow(fileHash + ":SND_STORAGE_INFO_SP:")
	p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, req, header.ReqFileStorageInfo)
}

func ClearFileInfoAndDownloadTask(ctx context.Context, fileHash string, fileReqId string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		task.DownloadFileMap.Delete(fileHash + fileReqId)
		task.DeleteDownloadTask(fileHash, setting.WalletAddress, "")
		req := &protos.ReqClearDownloadTask{
			WalletAddress: setting.WalletAddress,
			FileHash:      fileHash,
			P2PAddress:    p2pserver.GetP2pServer(ctx).GetP2PAddress(),
		}
		p2pServer := p2pserver.GetP2pServer(ctx)
		_ = p2pServer.SendMessage(ctx, p2pServer.GetPpConn(), req, header.ReqClearDownloadTask)
		_, _ = w.Write([]byte("ok"))
	} else {
		if w != nil {
			_, _ = w.Write(httpserv.NewJson(nil, setting.FAILCode, "login first").ToBytes())
		}
	}
}

func ReqClearDownloadTask(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqClearDownloadTask
	if err := VerifyMessage(ctx, header.ReqClearDownloadTask, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if requests.UnmarshalData(ctx, &target) {
		task.DeleteDownloadTask(target.WalletAddress, target.WalletAddress, "")
	}
}

// ReqFileStorageInfo  P-PP , PP-SP
func ReqFileStorageInfo(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqFileStorageInfo
	if err := VerifyMessage(ctx, header.ReqFileStorageInfo, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
	}
	utils.Log("pp get ReqFileStorageInfo directly transfer to SP")
	p2pserver.GetP2pServer(ctx).TransferSendMessageToSPServer(ctx, core.MessageFromContext(ctx))
}

// RspFileStorageInfo SP-PP , PP-P
func RspFileStorageInfo(ctx context.Context, conn core.WriteCloser) {
	// PP check whether itself is the storage PP, if not transfer
	pp.Log(ctx, "get，RspFileStorageInfo")
	var target protos.RspFileStorageInfo
	if err := VerifyMessage(ctx, header.RspFileStorageInfo, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	// SPAM check
	if time.Now().Unix()-target.TimeStamp > setting.SpamThresholdSpSignLatency {
		pp.ErrorLog(ctx, "sp's upload file response was expired")
		return
	}

	if target.Result.State == protos.ResultState_RES_FAIL {
		pp.ErrorLog(ctx, "Received fail massage from sp: ", target.Result.Msg)
		return
	}
	metrics.DownloadPerformanceLogNow(target.FileHash + ":RCV_STORAGE_INFO_SP:")

	newTarget := &protos.RspFileStorageInfo{
		VisitCer:      target.VisitCer,
		P2PAddress:    target.P2PAddress,
		WalletAddress: target.WalletAddress,
		SliceInfo:     target.SliceInfo,
		FileHash:      target.FileHash,
		FileName:      target.FileName,
		Result:        target.Result,
		ReqId:         target.ReqId,
		SavePath:      target.SavePath,
		FileSize:      target.FileSize,
		RestAddress:   target.RestAddress,
		NodeSign:      target.NodeSign,
		SpP2PAddress:  target.SpP2PAddress,
		EncryptionTag: target.EncryptionTag,
		TaskId:        target.TaskId,
		TimeStamp:     target.TimeStamp,
	}

	fileReqId := core.GetRemoteReqId(ctx)
	newTarget.ReqId = fileReqId
	pp.DebugLog(ctx, "file hash, reqid:", target.FileHash, fileReqId)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		task.CleanDownloadFileAndConnMap(ctx, target.FileHash, fileReqId)
		task.DownloadFileMap.Store(target.FileHash+fileReqId, newTarget)
		task.AddDownloadTask(newTarget)
		if utils.IsVideoStream(target.FileHash) {
			file.SetRemoteFileResult(target.FileHash+fileReqId, rpc.Result{Return: rpc.DOWNLOAD_OK, FileHash: target.FileHash})
			return
		}
		DownloadFileSlice(ctx, newTarget, fileReqId)
	} else {
		file.SetRemoteFileResult(target.FileHash+fileReqId, rpc.Result{Return: rpc.FILE_REQ_FAILURE})
		pp.Log(ctx, "failed to download，", target.Result.Msg)
	}
}

func GetFileReplicaInfo(ctx context.Context, path string, replicaIncreaseNum uint32) {
	utils.DebugLog("GetFileReplicaInfo")

	if !setting.CheckLogin() {
		return
	}
	if len(path) < setting.DownloadPathMinLen {
		utils.DebugLog("invalid path length")
		return
	}
	_, _, fileHash, _, err := datamesh.ParseFileHandle(path)
	if err != nil {
		pp.ErrorLog(ctx, "please input correct file link, eg: sdm://address/fileHash|filename(optional)")
		return
	}
	pp.DebugLog(ctx, "path:", path)

	nowSec := time.Now().Unix()
	// sign the wallet signature by wallet private key
	wsignMsg := utils.GetFileReplicaInfoWalletSignMessage(fileHash, setting.WalletAddress, nowSec)
	wsign, err := types.BytesToAccPriveKey(setting.WalletPrivateKey).Sign([]byte(wsignMsg))
	if err != nil {
		return
	}

	req := requests.ReqFileReplicaInfo(path, setting.WalletAddress, p2pserver.GetP2pServer(ctx).GetP2PAddress(), replicaIncreaseNum,
		setting.WalletPublicKey, wsign, nowSec)
	p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, req, header.ReqFileReplicaInfo)
}

func RspFileReplicaInfo(ctx context.Context, conn core.WriteCloser) {
	pp.Log(ctx, "get，RspGetFileReplicaInfo")
	var target protos.RspFileReplicaInfo
	if err := VerifyMessage(ctx, header.RspFileReplicaInfo, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	if target.Result.State == protos.ResultState_RES_FAIL {
		pp.ErrorLog(ctx, "Received fail massage from sp: ", target.Result.Msg)
		return
	}

	pp.Log(ctx, "file hash", target.FileHash)
	pp.Log(ctx, "file replicas", target.Replicas)
	pp.Log(ctx, "file expected replicas", target.ExpectedReplicas)
}

// GetFileStatus checks if the specified file is currently being uploaded. If it isn't, it queries the file status from SP to know if the upload succeeded or failed
func GetFileStatus(ctx context.Context, fileHash, walletAddr string, walletPubkey, walletSign []byte, reqTime int64) *protos.RspFileStatus {
	if value, found := task.UploadFileTaskMap.Load(fileHash); found {
		uploadTask := value.(*task.UploadFileTask)
		rsp := &protos.RspFileStatus{
			Result:   &protos.Result{State: protos.ResultState_RES_SUCCESS},
			FileHash: fileHash,
			State:    protos.FileUploadState_UPLOADING,
		}
		if uploadTask.IsFatal() != nil {
			rsp.State = protos.FileUploadState_FAILED
			rsp.Result.Msg = uploadTask.IsFatal().Error()
		}
		return rsp
	}

	// If not, send req to sp
	req := requests.ReqFileStatus(fileHash, walletAddr, walletPubkey, walletSign, reqTime)
	p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, req, header.ReqFileStatus)
	return nil
}

func RspFileStatus(ctx context.Context, conn core.WriteCloser) {
	pp.Log(ctx, "get RspFileStatus")
	var target protos.RspFileStatus
	if err := VerifyMessage(ctx, header.RspFileStatus, &target); err != nil {
		pp.ErrorLog(ctx, "failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	fileStatusResult := &rpc.FileStatusResult{
		Return:          rpc.SUCCESS,
		FileUploadState: target.State,
		UserHasFile:     target.UserHasFile,
		Replicas:        target.Replicas,
	}

	if target.Result.State == protos.ResultState_RES_FAIL {
		fileStatusResult.Error = target.Result.Msg
	}

	if bytes, err := json.Marshal(fileStatusResult); err == nil {
		pp.Logf(ctx, "RspFileStatus: %v", string(bytes))
	}

	fileReqId := core.GetRemoteReqId(ctx)
	file.SetGetFileStatusDone(target.FileHash+fileReqId, fileStatusResult)
}
