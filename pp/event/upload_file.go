package event

// Author j
import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/metrics"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/ipfs"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/types"
)

var (
	isCover          bool
	requestUploadMap = &sync.Map{}
)

// MigrateIpfsFile migrate ipfs file to sds
func MigrateIpfsFile(ctx context.Context, cid, fileName string) {
	filePath, err := ipfs.GetFile(ctx, cid, fileName)
	if err != nil {
		pp.ErrorLog(ctx, "failed to fetch the file from ipfs: ", err)
		return
	}
	RequestUploadFile(ctx, filePath, false, nil)
}

// RequestUploadFile request to SP for upload file
func RequestUploadFile(ctx context.Context, path string, isEncrypted bool, _ http.ResponseWriter) {
	pp.DebugLog(ctx, "______________path", path)
	if !setting.CheckLogin() {
		return
	}

	fileHash := file.GetFileHash(path, "")
	metrics.UploadPerformanceLogNow(fileHash + ":RCV_CMD_START:")

	isFile, err := file.IsFile(path)
	if err != nil {
		pp.ErrorLog(ctx, err)
		return
	}
	if !isFile {
		var target string
		if path[len(path)-1:] == "/" {
			target = path[:len(path)-1] + ".tar.zst"
		} else {
			target = path + ".tar.zst"
		}
		file.CreateTarWithZstd(path, target)
		utils.DebugLog("new path:", target)
		path = target
	}
	p := requests.RequestUploadFileData(ctx, path, "", false, false, isEncrypted)
	if err = ReqGetWalletOzForUpload(ctx, setting.WalletAddress, task.LOCAL_REQID, p); err != nil {
		pp.ErrorLog(ctx, err)
	}
}

func RequestUploadStream(ctx context.Context, path string) {
	pp.DebugLog(ctx, "______________path", path)
	if !setting.CheckLogin() {
		return
	}
	isFile, err := file.IsFile(path)
	if err != nil {
		pp.ErrorLog(ctx, err)
		return
	}
	if isFile {
		p := requests.RequestUploadFileData(ctx, path, "", false, true, false)
		if err = ReqGetWalletOzForUpload(ctx, setting.WalletAddress, task.LOCAL_REQID, p); err != nil {
			pp.ErrorLog(ctx, err)
		}
		return
	} else {
		pp.ErrorLog(ctx, "the provided path indicates a directory, not a file")
		return
	}
}

func ScheduleReqBackupStatus(ctx context.Context, fileHash string) {
	time.AfterFunc(5*time.Minute, func() {
		ReqBackupStatus(ctx, fileHash)
	})
}

func ReqBackupStatus(ctx context.Context, fileHash string) {
	p := &protos.ReqBackupStatus{
		FileHash: fileHash,
		Address:  setting.GetPPInfo(),
	}
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, p, header.ReqFileBackupStatus)
}

// RspUploadFile response of upload file event, SP -> upgrader, upgrader -> dest PP
func RspUploadFile(ctx context.Context, _ core.WriteCloser) {
	pp.DebugLog(ctx, "get RspUploadFile")
	target := &protos.RspUploadFile{}
	if !requests.UnmarshalData(ctx, target) {
		pp.ErrorLog(ctx, "unmarshal error")
		return
	}

	metrics.UploadPerformanceLogNow(target.FileHash + ":RCV_RSP_UPLOAD_SP")

	// SPAM check
	if time.Now().Unix()-target.TimeStamp > setting.SPAM_THRESHOLD_SP_SIGN_LATENCY {
		pp.ErrorLog(ctx, "sp's upload file response was expired")
		return
	}

	// verify if this is the response from earlier request from me
	if fh, found := requestUploadMap.LoadAndDelete(requests.GetReqIdFromMessage(ctx)); !found || target.FileHash != fh {
		pp.ErrorLog(ctx, "file upload response doesn't match the request")
		return
	}

	// upload file to PP based on the PP info provided by SP
	if target.Result == nil {
		pp.ErrorLog(ctx, "target.Result is nil")
		return
	}

	if target.Result.State != protos.ResultState_RES_SUCCESS {
		if strings.Contains(target.Result.Msg, "Same file with the name") {
			pp.Log(ctx, target.Result.Msg)
		} else {
			pp.Log(ctx, "upload failed: ", target.Result.Msg)
		}

		if file.IsFileRpcRemote(target.FileHash) {
			file.SetRemoteFileResult(target.FileHash, rpc.Result{Return: target.Result.Msg})
		} else {
			file.ClearFileMap(target.FileHash)
		}
		return
	}

	spP2pPubkey, err := requests.GetSpPubkey(target.SpP2PAddress)
	if err != nil {
		pp.ErrorLog(ctx, "failed to get sp pubkey")
		return
	}

	// verify sp address
	if !types.VerifyP2pAddrBytes(spP2pPubkey, target.SpP2PAddress) {
		pp.ErrorLog(ctx, "failed verifying sp's p2p address")
		return
	}

	// verify sp node signature
	nodeSign := target.NodeSign
	target.NodeSign = nil
	msg, err := utils.GetRspUploadFileSpNodeSignMessage(target)
	if err != nil {
		pp.ErrorLog(ctx, "failed calculating signature from message")
		return
	}
	if !types.VerifyP2pSignBytes(spP2pPubkey, nodeSign, msg) {
		pp.ErrorLog(ctx, "failed verifying signature from sp")
		return
	}
	target.NodeSign = nodeSign

	if len(target.Slices) != 0 {
		go startUploadTask(ctx, target)
	} else {
		pp.Log(ctx, "file upload successful！  fileHash", target.FileHash)
		//var p float32 = 100
		//ProgressMap.Store(target.FileHash, p)
		task.UploadProgressMap.Delete(target.FileHash)

		if file.IsFileRpcRemote(target.FileHash) {
			file.SetRemoteFileResult(target.FileHash, rpc.Result{Return: rpc.SUCCESS})
		}
	}
	if isCover {
		pp.DebugLog(ctx, "is_cover")
	}
}

func RspBackupStatus(ctx context.Context, _ core.WriteCloser) {
	pp.DebugLog(ctx, "get RspBackupStatus")
	target := &protos.RspBackupStatus{}
	if !requests.UnmarshalData(ctx, target) {
		pp.ErrorLog(ctx, "unmarshal error")
		return
	}

	// SPAM check
	if time.Now().Unix()-target.TimeStamp > setting.SPAM_THRESHOLD_SP_SIGN_LATENCY {
		pp.ErrorLog(ctx, "sp's backup file response was expired,", time.Now().Unix(), ",", target.TimeStamp)
		return
	}

	spP2pPubkey, err := requests.GetSpPubkey(target.SpP2PAddress)
	if err != nil {
		pp.ErrorLog(ctx, "failed to get sp pubkey")
		return
	}

	// verify sp address
	if !types.VerifyP2pAddrBytes(spP2pPubkey, target.SpP2PAddress) {
		pp.ErrorLog(ctx, "failed verifying sp's p2p address")
		return
	}

	// verify sp node signature
	nodeSign := target.NodeSign
	target.NodeSign = nil
	msg, err := utils.GetRspBackupFileSpNodeSignMessage(target)
	if err != nil {
		pp.ErrorLog(ctx, "failed calculating signature from message")
		return
	}
	if !types.VerifyP2pSignBytes(spP2pPubkey, nodeSign, msg) {
		pp.ErrorLog(ctx, "failed verifying signature from sp")
		return
	}
	target.NodeSign = nodeSign

	if target.Result.State == protos.ResultState_RES_FAIL {
		pp.Log(ctx, "Backup status check failed", target.Result.Msg)
		return
	}

	pp.Logf(ctx, "Backup status for file %s: the number of replica is %d", target.FileHash, target.Replicas)
	if target.DeleteOriginTmp {
		pp.Logf(ctx, "Backup is finished for file %s, delete all the temporary slices", target.FileHash)
		file.DeleteTmpFileSlices(ctx, target.FileHash)
		return
	}

	if len(target.Slices) == 0 {
		ScheduleReqBackupStatus(ctx, target.FileHash)
		return
	}

	pp.Logf(ctx, "Start re-uploading slices for the file  %s", target.FileHash)
	totalSize := int64(0)
	for _, slice := range target.Slices {
		totalSize += int64(slice.GetSliceSize())
	}
	uploadTask := task.CreateBackupFileTask(target)
	p2pserver.GetP2pServer(ctx).CleanUpConnMap("upload#" + target.FileHash)
	task.UploadFileTaskMap.Store(target.FileHash, uploadTask)

	p := &task.UploadProgress{
		Total:     totalSize,
		HasUpload: 0,
	}
	task.UploadProgressMap.Store(target.FileHash, p)

	// Start uploading
	startUploadingFileSlices(ctx, target.FileHash)
}

func startUploadTask(ctx context.Context, target *protos.RspUploadFile) {
	// Create upload task
	uploadTask := task.CreateUploadFileTask(target)

	p2pserver.GetP2pServer(ctx).CleanUpConnMap(target.FileHash)
	task.UploadFileTaskMap.Store(target.FileHash, uploadTask)

	// Video Hls transformation and upload progress update
	var streamTotalSize int64
	var hlsInfo file.HlsInfo
	if target.IsVideoStream {
		if file.IsFileRpcRemote(uploadTask.RspUploadFile.FileHash) {
			remotePath := strings.Split(file.GetFilePath(target.FileHash), ":")
			fileName := remotePath[len(remotePath)-1]
			file.VideoToHls(ctx, target.FileHash, filepath.Join(setting.GetRootPath(), file.TEMP_FOLDER, target.FileHash, fileName))
		} else {
			file.VideoToHls(ctx, target.FileHash, file.GetFilePath(target.FileHash))
		}

		if hlsInfo, err := file.GetHlsInfo(target.FileHash, uint64(len(target.Slices))); err != nil {
			pp.ErrorLog(ctx, "Hls transformation failed: ", err)
			return
		} else {
			streamTotalSize = hlsInfo.TotalSize
			file.HlsInfoMap[target.FileHash] = hlsInfo
		}
	}
	if prg, ok := task.UploadProgressMap.Load(target.FileHash); ok {
		progress := prg.(*task.UploadProgress)
		if target.IsVideoStream {
			jsonStr, _ := json.Marshal(hlsInfo)
			progress.Total = streamTotalSize + int64(len(jsonStr))
		}
		progress.HasUpload = (target.TotalSlice - int64(len(target.Slices))) * setting.MAX_SLICE_SIZE
	}

	// Start uploading
	startUploadingFileSlices(ctx, target.FileHash)
}

func startUploadingFileSlices(ctx context.Context, fileHash string) {
	value, ok := task.UploadFileTaskMap.Load(fileHash)
	if !ok {
		pp.ErrorLogf(ctx, "File upload task cannot be found for file %v", fileHash)
		return
	}
	fileTask := value.(*task.UploadFileTask)

	// Send signals to start slice uploads
	fileTask.SignalNewDestinations()

	err := waitForUploadFinished(ctx, fileTask)
	if err != nil {
		pp.ErrorLog(ctx, "File upload task will be cancelled: ", utils.FormatError(err))
		return
	}

	task.UploadFileTaskMap.Delete(fileHash)

	if fileTask.RspUploadFile.IsVideoStream {
		file.DeleteTmpHlsFolder(ctx, fileHash)
	}
}

func waitForUploadFinished(ctx context.Context, uploadTask *task.UploadFileTask) error {
	for {
		// Wait until a new destination can be uploaded to, or until some time has passed
		select {
		case keepGoing, ok := <-uploadTask.UpChan:
			if !ok {
				return nil
			}
			if !keepGoing {
				continue
			}
		case <-time.After(task.UPLOAD_TIMER_INTERVAL * time.Second):
		}

		if uploadTask.IsFinished() {
			return nil
		}

		if err := uploadTask.IsFatal(); err != nil {
			return err
		}

		// Report slice failures to SP to get assigned new slice destinations
		slicesToReDownload, failedSlices := uploadTask.SliceFailuresToReport()
		if len(slicesToReDownload) > 0 {
			if !uploadTask.CanRetry() {
				return errors.New("max upload retry count reached")
			}
			p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqUploadSlicesWrong(uploadTask, uploadTask.RspUploadFile.SpP2PAddress, slicesToReDownload, failedSlices), header.ReqUploadSlicesWrong)
		}

		// Start uploading to next destination
		nextDestination := uploadTask.NextDestination()
		if nextDestination != nil {
			go uploadSlicesToDestination(ctx, uploadTask, nextDestination)
		}
	}
}

func uploadSlicesToDestination(ctx context.Context, uploadTask *task.UploadFileTask, destination *task.SlicesPerDestination) {
	defer func() {
		uploadTask.UpChan <- true
	}()
	for _, slice := range destination.Slices {
		if uploadTask.FatalError != nil {
			return
		}

		var uploadSliceTask *task.UploadSliceTask
		var err error
		switch uploadTask.Type {
		case protos.UploadType_NEW_UPLOAD:
			uploadSliceTask, err = task.CreateUploadSliceTask(ctx, slice, destination.PpInfo, uploadTask)
			if err != nil {
				slice.SetError(err, true, uploadTask)
				return
			}
			pp.DebugLogf(ctx, "starting to upload slice %v for file %v", slice.Slice.SliceNumber, uploadTask.RspUploadFile.FileHash)
			err = UploadFileSlice(ctx, uploadSliceTask)
			if err != nil {
				slice.SetError(err, false, uploadTask)
				return
			}
		case protos.UploadType_BACKUP:
			uploadSliceTask, err = task.GetReuploadSliceTask(ctx, slice, destination.PpInfo, uploadTask)
			if err != nil {
				slice.SetError(err, true, uploadTask)
				return
			}
			pp.DebugLogf(ctx, "starting to backup slice %v for file %v", slice.Slice.SliceNumber, uploadTask.RspUploadFile.FileHash)
			err = BackupFileSlice(ctx, uploadSliceTask)
			if err != nil {
				slice.SetError(err, false, uploadTask)
				return
			}
		}

		slice.SetStatus(task.SLICE_STATUS_FINISHED, uploadTask)
	}
}

// UploadPause
func UploadPause(ctx context.Context, fileHash, reqID string, w http.ResponseWriter) {
	p2pserver.GetP2pServer(ctx).RangeCachedConn("upload#"+fileHash, func(k, v interface{}) bool {
		conn := v.(*cf.ClientConn)
		conn.ClientClose(true)
		pp.DebugLog(ctx, "UploadPause", conn)
		return true
	})
	p2pserver.GetP2pServer(ctx).CleanUpConnMap(fileHash)
	task.UploadFileTaskMap.Delete(fileHash)
	task.UploadProgressMap.Delete(fileHash)
}

func StoreUploadReqId(ctx context.Context, fileHash string) context.Context {
	ctx = core.CreateContextWithReqId(ctx, requests.GetReqIdFromMessage(ctx))
	requestUploadMap.Store(core.GetReqIdFromContext(ctx), fileHash)
	return ctx
}
