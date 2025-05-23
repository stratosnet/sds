package event

// Author j
import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/crypto"
	"github.com/stratosnet/sds/framework/msg/header"
	fwtypes "github.com/stratosnet/sds/framework/types"
	fwutils "github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/framework/utils/httpserv"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/metrics"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/sds-msg/protos"
	msgutils "github.com/stratosnet/sds/sds-msg/utils"
)

// GetFileStorageInfo p to pp. The downloader is assumed the default wallet of this node, if this function is invoked.
func GetFileStorageInfo(ctx context.Context, path, savePath, saveAs string, w http.ResponseWriter) {
	fwutils.DebugLog("GetFileStorageInfo")

	if !setting.CheckLogin() {
		if w != nil {
			_, _ = w.Write(httpserv.NewJson(nil, setting.FAILCode, "login first").ToBytes())
		}
		return
	}
	if len(path) < setting.DownloadPathMinLen {
		fwutils.DebugLog("invalid path length")
		return
	}
	_, walletAddress, fileHash, _, err := fwtypes.ParseFileHandle(path)
	if err != nil {
		pp.ErrorLog(ctx, "please input correct download link, eg: sdm://address/fileHash|filename(optional)")
		if w != nil {
			_, _ = w.Write(httpserv.NewJson(nil, setting.FAILCode, "please input correct download link, eg:  sdm://address/fileHash|filename(optional)").ToBytes())
		}
		return
	}
	if walletAddress != setting.WalletAddress {
		errMsg := "only the file owner is allowed to download via sdm url"
		pp.ErrorLog(ctx, errMsg)
		if w != nil {
			_, _ = w.Write(httpserv.NewJson(nil, setting.FAILCode, errMsg).ToBytes())
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
	wsignMsg := msgutils.GetFileDownloadWalletSignMessage(fileHash, setting.WalletAddress, "", nowSec) // need to retrieve sn first
	wsign, err := setting.WalletPrivateKey.Sign([]byte(wsignMsg))
	if err != nil {
		return
	}
	req := requests.ReqFileStorageInfoData(ctx, path, savePath, saveAs, setting.WalletAddress, setting.WalletPublicKey.Bytes(), wsign, nil, nowSec)
	metrics.DownloadPerformanceLogNow(fileHash + ":SND_STORAGE_INFO_SP:")
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, req, header.ReqFileStorageInfo)
}

func ClearFileInfoAndDownloadTask(ctx context.Context, fileHash string, fileReqId string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		task.DownloadFileMap.Delete(fileHash + fileReqId)
		task.DeleteDownloadTask(fileHash, setting.WalletAddress, "")

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
		fwutils.ErrorLog("failed verifying the message, ", err.Error())
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
		fwutils.ErrorLog("failed verifying the message, ", err.Error())
	}
	fwutils.Log("pp get ReqFileStorageInfo directly transfer to SP")
	p2pserver.GetP2pServer(ctx).TransferSendMessageToSPServer(ctx, core.MessageFromContext(ctx))
}

// RspFileStorageInfo SP-PP , PP-P
func RspFileStorageInfo(ctx context.Context, conn core.WriteCloser) {
	// PP check whether itself is the storage PP, if not transfer
	var target protos.RspFileStorageInfo
	if err := VerifyMessage(ctx, header.RspFileStorageInfo, &target); err != nil {
		fwutils.ErrorLog("failed verifying the message, ", err.Error())
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

	fileReqId := core.GetRemoteReqId(ctx)
	rpcRequested := !strings.HasPrefix(fileReqId, task.LOCAL_REQID)
	if target.Result.State == protos.ResultState_RES_FAIL {
		task.DownloadResult(ctx, target.FileHash, false, "failed ReqFileStorageInfo, "+target.Result.Msg)
		file.SetDownloadSliceResult(target.FileHash+fileReqId, &rpc.Result{Return: rpc.FILE_REQ_FAILURE, Detail: "failed ReqFileStorageInfo" + target.Result.Msg})
		return
	}
	metrics.DownloadPerformanceLogNow(target.FileHash + ":RCV_STORAGE_INFO_SP:")

	newTarget := &target
	newTarget.ReqId = fileReqId
	task.CleanDownloadFileAndConnMap(ctx, target.FileHash, fileReqId)
	task.DownloadFileMap.Store(target.FileHash+fileReqId, newTarget)
	task.AddDownloadTask(newTarget)

	var slice *protos.DownloadSliceInfo
	var slices []file.DownloadSlice
	for _, slice = range target.SliceInfo {
		s := file.DownloadSlice{
			SliceHash:       slice.SliceStorageInfo.SliceHash,
			SliceFileOffset: slice.SliceOffset.SliceOffsetStart,
			SliceSize:       slice.SliceStorageInfo.SliceSize,
		}
		slices = append(slices, s)
	}

	file.SetDownloadFileInfo(target.FileHash, file.DownloadFile{
		FileHash: target.FileHash,
		FileSize: target.FileSize,
		FileName: target.FileName,
		Slices:   slices,
	})
	file.SetDownloadSliceResult(target.FileHash+fileReqId, &rpc.Result{Return: rpc.DOWNLOAD_OK})
	if crypto.IsVideoStream(target.FileHash) {
		_ = file.SetRemoteFileResult(target.FileHash+fileReqId, rpc.Result{Return: rpc.DOWNLOAD_OK, FileHash: target.FileHash})
		return
	}
	if !rpcRequested {
		file.StartLocalDownload(target.FileHash)
	}
	DownloadFileSlices(ctx, newTarget, fileReqId)
}

func GetFileReplicaInfo(ctx context.Context, path string, replicaIncreaseNum uint32) {
	fwutils.DebugLog("GetFileReplicaInfo")

	if !setting.CheckLogin() {
		return
	}
	if len(path) < setting.DownloadPathMinLen {
		fwutils.DebugLog("invalid path length")
		return
	}
	pp.DebugLog(ctx, "path:", path)

	nowSec := time.Now().Unix()
	spP2pAddress := p2pserver.GetP2pServer(ctx).GetP2PAddress().String()
	req := requests.ReqFileReplicaInfo(path, setting.WalletAddress, spP2pAddress, replicaIncreaseNum, setting.WalletPublicKey.Bytes(), nil, nowSec)
	if err := ReqGetWalletOzForReplicas(ctx, setting.WalletAddress, task.LOCAL_REQID, req); err != nil {
		pp.ErrorLog(ctx, err)
		return
	}
}

func RspFileReplicaInfo(ctx context.Context, conn core.WriteCloser) {
	pp.Log(ctx, "get，RspGetFileReplicaInfo")
	var target protos.RspFileReplicaInfo
	if err := VerifyMessage(ctx, header.RspFileReplicaInfo, &target); err != nil {
		fwutils.ErrorLog("failed verifying the message, ", err.Error())
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

	taskId := ""
	if id, found := task.UploadTaskIdMap.Load(fileHash); found {
		taskId = id.(string)
	}

	// If not, send req to sp
	req := requests.ReqFileStatus(fileHash, walletAddr, taskId, walletPubkey, walletSign, reqTime)
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, req, header.ReqFileStatus)
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

	if target.Result.State == protos.ResultState_RES_SUCCESS && target.Replicas == 0 {
		errMsg := fmt.Sprintf("No available replicas remains for the file %s, re-upload it if you still want to use it, "+
			"please be kindly noted that you won't be charged for re-uploading this file", target.FileHash)
		fileStatusResult.Error = errMsg
	}

	if bytes, err := json.Marshal(fileStatusResult); err == nil {
		pp.Logf(ctx, "RspFileStatus: %v", string(bytes))
	}

	fileReqId := core.GetRemoteReqId(ctx)
	file.SetGetFileStatusDone(target.FileHash+fileReqId, fileStatusResult)
}
