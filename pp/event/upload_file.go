package event

// Author j
import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/httpserv"
)

//var m *sync.WaitGroup
var isCover bool

// RequestUploadCoverImage RequestUploadCoverImage
func RequestUploadCoverImage(pathStr, reqID string, w http.ResponseWriter) {
	isCover = true
	tmpString, err := utils.ImageCommpress(pathStr)
	utils.DebugLog("reqID", reqID)
	if err != nil {
		utils.ErrorLog(err)
		if w != nil {
			_, _ = w.Write(httpserv.NewJson(nil, setting.FAILCode, "compress image failed").ToBytes())
		}
		return
	}
	p := requests.RequestUploadFileData(tmpString, "", reqID, setting.WalletAddress, true, false, false)
	peers.SendMessageToSPServer(p, header.ReqUploadFile)
	storeResponseWriter(reqID, w)
}

// RequestUploadFile request to SP for upload file
func RequestUploadFile(path, reqID string, isEncrypted bool, _ http.ResponseWriter) {
	utils.DebugLog("______________path", path)
	if !setting.CheckLogin() {
		return
	}

	isFile, err := file.IsFile(path)
	if err != nil {
		utils.ErrorLog(err)
		return
	}
	if isFile {
		p := requests.RequestUploadFileData(path, "", reqID, setting.WalletAddress, false, false, isEncrypted)
		peers.SendMessageToSPServer(p, header.ReqUploadFile)
		return
	}

	// is directory
	utils.DebugLog("this is a directory, not file")
	file.GetAllFiles(path)
	for {
		select {
		case pathString := <-setting.UpChan:
			utils.DebugLog("path string == ", pathString)
			p := requests.RequestUploadFileData(pathString, "", reqID, setting.WalletAddress, false, false, isEncrypted)
			peers.SendMessageToSPServer(p, header.ReqUploadFile)
		default:
			return
		}
	}
}

func RequestUploadStream(path, reqID string, _ http.ResponseWriter) {
	utils.DebugLog("______________path", path)
	if !setting.CheckLogin() {
		return
	}
	isFile, err := file.IsFile(path)
	if err != nil {
		utils.ErrorLog(err)
		return
	}
	if isFile {
		p := requests.RequestUploadFileData(path, "", reqID, setting.WalletAddress, false, true, false)
		if p != nil {
			peers.SendMessageToSPServer(p, header.ReqUploadFile)
		}
		return
	} else {
		utils.ErrorLog("the provided path indicates a directory, not a file")
		return
	}
}

func ScheduleReqBackupStatus(fileHash string) {
	time.AfterFunc(5*time.Minute, func() {
		ReqBackupStatus(fileHash)
	})
}

func ReqBackupStatus(fileHash string) {
	p := &protos.ReqBackupStatus{
		FileHash:      fileHash,
		P2PAddress:    setting.P2PAddress,
		WalletAddress: setting.WalletAddress,
	}
	peers.SendMessageToSPServer(p, header.ReqFileBackupStatus)
}

// RspUploadFile response of upload file event
func RspUploadFile(ctx context.Context, _ core.WriteCloser) {
	utils.DebugLog("get RspUploadFile")
	target := &protos.RspUploadFile{}
	if !requests.UnmarshalData(ctx, target) {
		utils.ErrorLog("unmarshal error")
		return
	}

	// upload file to PP based on the PP info provided by SP
	if target.Result == nil {
		utils.ErrorLog("target.Result is nil")
		return
	}

	if target.Result.State != protos.ResultState_RES_SUCCESS {
		if strings.Contains(target.Result.Msg, "Same file with the name") {
			utils.Log(target.Result.Msg)
		} else {
			utils.Log("upload failed: ", target.Result.Msg)
		}

		if file.IsFileRpcRemote(target.FileHash) {
			file.SetRemoteFileResult(target.FileHash, rpc.Result{Return: target.Result.Msg})
		} else {
			file.ClearFileMap(target.FileHash)
		}
		return
	}

	if len(target.PpList) != 0 {
		go startUploadTask(target)
	} else {
		utils.Log("file upload successful！  fileHash", target.FileHash)
		var p float32 = 100
		ProgressMap.Store(target.FileHash, p)
		task.UploadProgressMap.Delete(target.FileHash)

		if file.IsFileRpcRemote(target.FileHash) {
			file.SetRemoteFileResult(target.FileHash, rpc.Result{Return: rpc.SUCCESS})
		}
	}
	if isCover {
		utils.DebugLog("is_cover", target.ReqId)
		putData(target.ReqId, HTTPUploadFile, target)
	}
}

func RspBackupStatus(ctx context.Context, _ core.WriteCloser) {
	utils.DebugLog("get RspBackupStatus")
	target := &protos.RspBackupStatus{}
	if !requests.UnmarshalData(ctx, target) {
		utils.ErrorLog("unmarshal error")
		return
	}

	if target.Result.State == protos.ResultState_RES_FAIL {
		utils.Log("Backup status check failed", target.Result.Msg)
		return
	}

	utils.Logf("Backup status for file %s: the number of replica is %d", target.FileHash, target.Replicas)
	if target.DeleteOriginTmp {
		utils.Logf("Backup is finished for file %s, delete all the temporary slices", target.FileHash)
		file.DeleteTmpFileSlices(target.FileHash)
		return
	}

	if len(target.PpList) == 0 {
		ScheduleReqBackupStatus(target.FileHash)
		return
	}

	utils.Logf("Start re-uploading slices for the file  %s", target.FileHash)
	totalSize := int64(0)
	for _, slice := range target.PpList {
		totalSize += int64(slice.GetSliceSize())
	}
	uploadTask := task.CreateUploadFileTask(target.FileHash, target.TaskId, target.SpP2PAddress, false, false, target.PpList, protos.UploadType_BACKUP)
	task.CleanUpConnMap(target.FileHash)
	task.UploadFileTaskMap.Store(target.FileHash, uploadTask)

	p := &task.UploadProgress{
		Total:     totalSize,
		HasUpload: 0,
	}
	task.UploadProgressMap.Store(target.FileHash, p)

	// Start uploading
	startUploadingFileSlices(target.FileHash)
}

func startUploadTask(target *protos.RspUploadFile) {
	var slices []*protos.SliceHashAddr
	for _, slice := range target.PpList {
		slices = append(slices, &protos.SliceHashAddr{
			SliceHash:   "",
			SliceNumber: slice.SliceNumber,
			SliceSize:   slice.SliceOffset.SliceOffsetEnd - slice.SliceOffset.SliceOffsetStart,
			SliceOffset: slice.SliceOffset,
			PpInfo:      slice.PpInfo,
		})
	}

	// Create upload task
	uploadTask := task.CreateUploadFileTask(target.FileHash, target.TaskId, target.SpP2PAddress, target.IsEncrypted, target.IsVideoStream, slices, protos.UploadType_NEW_UPLOAD)
	task.CleanUpConnMap(target.FileHash)
	task.UploadFileTaskMap.Store(target.FileHash, uploadTask)

	// Video Hls transformation and upload progress update
	var streamTotalSize int64
	var hlsInfo file.HlsInfo
	if target.IsVideoStream {
		file.VideoToHls(target.FileHash)
		if hlsInfo, err := file.GetHlsInfo(target.FileHash, uint64(len(target.PpList))); err != nil {
			utils.ErrorLog("Hls transformation failed: ", err)
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
		progress.HasUpload = (target.TotalSlice - int64(len(target.PpList))) * 32 * 1024 * 1024
	}

	// Start uploading
	startUploadingFileSlices(target.FileHash)
}

func startUploadingFileSlices(fileHash string) {
	value, ok := task.UploadFileTaskMap.Load(fileHash)
	if !ok {
		utils.ErrorLogf("File upload task cannot be found for file %v", fileHash)
		return
	}
	fileTask := value.(*task.UploadFileTask)

	// Send signals to start slice uploads
	fileTask.SignalNewDestinations()

	err := waitForUploadFinished(fileTask)
	if err != nil {
		utils.ErrorLog("File upload task will be cancelled: ", utils.FormatError(err))
		return
	}

	task.UploadFileTaskMap.Delete(fileHash)

	if fileTask.IsVideoStream {
		file.DeleteTmpHlsFolder(fileHash)
	}
}

func waitForUploadFinished(uploadTask *task.UploadFileTask) error {
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
			peers.SendMessageToSPServer(requests.ReqUploadSlicesWrong(uploadTask, uploadTask.SpP2pAddress, slicesToReDownload, failedSlices), header.ReqUploadSlicesWrong)
		}

		// Start uploading to next destination
		nextDestination := uploadTask.NextDestination()
		if nextDestination != nil {
			go uploadSlicesToDestination(uploadTask, nextDestination)
		}
	}
}

func uploadSlicesToDestination(uploadTask *task.UploadFileTask, destination *task.SlicesPerDestination) {
	defer func() {
		uploadTask.UpChan <- true
	}()
	for _, slice := range destination.Slices {
		var uploadSliceTask *task.UploadSliceTask
		var err error
		switch uploadTask.Type {
		case protos.UploadType_NEW_UPLOAD:
			uploadSliceTask, err = task.CreateUploadSliceTask(slice, destination.PpInfo, uploadTask)
		case protos.UploadType_BACKUP:
			uploadSliceTask, err = task.GetReuploadSliceTask(slice, destination.PpInfo, uploadTask)
		}
		if err != nil {
			slice.SetError(err, true, uploadTask)
			return
		}

		utils.DebugLogf("starting to upload slice %v for file %v", slice.SliceNumber, uploadTask.FileHash)
		err = UploadFileSlice(uploadSliceTask, uploadTask.Sign)
		if err != nil {
			slice.SetError(err, false, uploadTask)
			return
		}
		slice.SetStatus(task.SLICE_STATUS_FINISHED, uploadTask)
	}
}

// UploadPause
func UploadPause(fileHash, reqID string, w http.ResponseWriter) {
	client.UpConnMap.Range(func(k, v interface{}) bool {
		if strings.HasPrefix(k.(string), fileHash) {
			conn := v.(*cf.ClientConn)
			conn.ClientClose()
			utils.DebugLog("UploadPause", conn)
		}
		return true
	})
	task.CleanUpConnMap(fileHash)
	task.UploadFileTaskMap.Delete(fileHash)
	task.UploadProgressMap.Delete(fileHash)
}
