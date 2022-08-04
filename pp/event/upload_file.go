package event

// Author j
import (
	"context"
	"encoding/json"
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

	} else if target.Result.State != protos.ResultState_RES_SUCCESS {
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

	} else if len(target.PpList) != 0 {
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
	var sliceAddrList []*protos.SliceNumAddr
	for _, sliceHashAddr := range target.PpList {
		sliceAddrList = append(sliceAddrList, &protos.SliceNumAddr{
			SliceNumber: sliceHashAddr.SliceNumber,
			SliceOffset: sliceHashAddr.SliceOffset,
			PpInfo:      sliceHashAddr.PpInfo,
		})
		totalSize += int64(sliceHashAddr.GetSliceSize())
	}
	uploadTask := &task.UploadFileTask{
		Slices:   nil,
		FileHash: target.FileHash,
		TaskID:   target.TaskId,
		UpChan:   make(chan bool, task.MAXSLICE),
	}
	task.CleanUpConnMap(target.FileHash)
	task.UploadFileTaskMap.Store(target.FileHash, uploadTask)

	p := &task.UploadProgress{
		Total:     totalSize,
		HasUpload: 0,
	}
	task.UploadProgressMap.Store(target.FileHash, p)

	for _, pp := range target.PpList {
		uploadSliceTask := task.GetReuploadSliceTask(pp, target.FileHash, target.TaskId, target.SpP2PAddress)
		if uploadSliceTask != nil {
			UploadFileSlice(uploadSliceTask, target.Sign)
		}
	}
	utils.DebugLog("all slices of the task have begun uploading")
	close(uploadTask.UpChan)
	task.UploadFileTaskMap.Delete(target.FileHash)
}

func startUploadTask(target *protos.RspUploadFile) {
	// Create upload task
	uploadTask := task.CreateUploadFileTask(target.FileHash, target.TaskId, target.PpList)
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
	startUploadingFileSlices(target)
}

func startUploadingFileSlices(target *protos.RspUploadFile) {
	value, ok := task.UploadFileTaskMap.Load(target.FileHash)
	if !ok {
		utils.ErrorLogf("File upload task cannot be found for file %v", target.FileHash)
		return
	}
	fileTask := value.(*task.UploadFileTask)

	// Send signals to start slice uploads
	go func() {
		numberOfDestinations := len(fileTask.Slices)
		if numberOfDestinations > task.MAXSLICE {
			numberOfDestinations = task.MAXSLICE
		}
		for i := 0; i < numberOfDestinations; i++ {
			fileTask.UpChan <- true
		}
	}()

	err := waitForUploadFinished(fileTask, target)
	if err != nil {
		// TODO: send upload failed to SP
		utils.ErrorLog(utils.FormatError(err))
		return
	}

	_, ok = <-fileTask.UpChan
	if ok {
		close(fileTask.UpChan)
	}
	task.UploadFileTaskMap.Delete(target.FileHash)

	if target.IsVideoStream {
		file.DeleteTmpHlsFolder(target.FileHash)
	}
}

func waitForUploadFinished(uploadTask *task.UploadFileTask, target *protos.RspUploadFile) error {
	for {
		// Wait until a new destination can be uploaded, or until some time has passed
		select {
		case keepGoing := <-uploadTask.UpChan:
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
			peers.SendMessageToSPServer(requests.ReqUploadSlicesWrong(uploadTask, slicesToReDownload, failedSlices), header.ReqUploadSlicesWrong)
		}

		// Start uploading to next destination
		nextDestination := uploadTask.NextDestination()
		if nextDestination != nil {
			go uploadSlicesToDestination(uploadTask, target, nextDestination)
		}
	}
}

func uploadSlicesToDestination(uploadTask *task.UploadFileTask, target *protos.RspUploadFile, destination *task.SlicesPerDestination) {
	for _, slice := range destination.Slices {
		protoSlice := &protos.SliceNumAddr{
			SliceNumber: slice.SliceNumber,
			SliceOffset: slice.SliceOffset,
			PpInfo:      destination.PpInfo,
		}
		uploadSliceTask, err := task.CreateUploadSliceTask(protoSlice, uploadTask.FileHash, uploadTask.TaskID, target.SpP2PAddress,
			target.IsVideoStream, target.IsEncrypted, uploadTask.FileCRC)
		if err != nil {
			slice.SetError(err, true, uploadTask)
			return
		}

		utils.DebugLogf("starting to upload slice %v for file %v", slice.SliceNumber, target.FileHash)
		err = UploadFileSlice(uploadSliceTask, target.Sign)
		if err != nil {
			slice.SetError(err, false, uploadTask)
			return
		}
		slice.SetStatus(task.SLICE_STATUS_FINISHED, uploadTask)
	}
}

func continueUpload(fileHash, taskID string) {
	utils.DebugLogf("continueUpload  fileHash = %v  taskID = %v", fileHash, taskID)
	if value, ok := task.UploadFileTaskMap.Load(fileHash); ok {
		uploadTask := value.(*task.UploadFileTask)
		uploadTask.UpChan <- true
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
