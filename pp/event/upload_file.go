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
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/httpserv"
	"github.com/stratosnet/sds/pp/api/rpc"
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
		}else {
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
			file.SetRemoteFileResult(target.FileHash, rpc.Result{Return:rpc.SUCCESS})
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
	taskING := &task.UpFileIng{
		UPING:    0,
		Slices:   sliceAddrList,
		FileHash: target.FileHash,
		TaskID:   target.TaskId,
		UpChan:   make(chan bool, task.MAXSLICE),
	}
	task.CleanUpConnMap(target.FileHash)
	task.UpIngMap.Store(target.FileHash, taskING)

	p := &task.UpProgress{
		Total:     totalSize,
		HasUpload: 0,
	}
	task.UploadProgressMap.Store(target.FileHash, p)

	for _, pp := range target.PpList {
		uploadTask := task.GetReuploadSliceTask(pp, target.FileHash, target.TaskId, target.SpP2PAddress)
		if uploadTask != nil {
			UploadFileSlice(uploadTask, target.Sign)
		}
	}
	utils.DebugLog("all slices of the task have begun uploading")
	close(taskING.UpChan)
	task.UpIngMap.Delete(target.FileHash)
}

// startUploadTask
func startUploadTask(target *protos.RspUploadFile) {
	// // create upload task
	slices := target.PpList
	taskING := &task.UpFileIng{
		UPING:    0,
		Slices:   slices,
		FileHash: target.FileHash,
		TaskID:   target.TaskId,
		UpChan:   make(chan bool, task.MAXSLICE),
		FileCRC:  utils.CalcFileCRC32(file.GetFilePath(target.FileHash)),
	}
	task.CleanUpConnMap(target.FileHash)
	task.UpIngMap.Store(target.FileHash, taskING)
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
		progress := prg.(*task.UpProgress)
		if target.IsVideoStream {
			jsonStr, _ := json.Marshal(hlsInfo)
			progress.Total = streamTotalSize + int64(len(jsonStr))
		}
		progress.HasUpload = (target.TotalSlice - int64(len(target.PpList))) * 32 * 1024 * 1024
	}
	go sendUploadFileSlice(target)
}

func up(ING *task.UpFileIng, target *protos.RspUploadFile) {
	for {
		select {
		case goon := <-ING.UpChan:
			if !goon {
				continue
			}

			if len(ING.Slices) == 0 {
				utils.DebugLog("all slices of the task have begun uploading")
				if _, ok := <-ING.UpChan; ok {
					close(ING.UpChan)
				}
				task.UpIngMap.Delete(target.FileHash)

				if target.IsVideoStream {
					file.DeleteTmpHlsFolder(target.FileHash)
				}

				return
			}
			pp := ING.Slices[0]
			utils.DebugLog("start upload!!!!!", pp.SliceNumber)
			uploadTask := task.GetUploadSliceTask(pp, ING.FileHash, ING.TaskID, target.SpP2PAddress,
				target.IsVideoStream, target.IsEncrypted, ING.FileCRC)
			if uploadTask == nil {
				continue
			}

			UploadFileSlice(uploadTask, target.Sign)
			ING.Slices = append(ING.Slices[:0], ING.Slices[0+1:]...)
		}
	}
}

func sendUploadFileSlice(target *protos.RspUploadFile) {
	ing, ok := task.UpIngMap.Load(target.FileHash)
	if !ok {
		utils.DebugLog("all slices of the task have begun uploading")
		return
	}
	ING := ing.(*task.UpFileIng)
	if len(ING.Slices) > task.MAXSLICE {
		go up(ING, target)
		for i := 0; i < task.MAXSLICE; i++ {
			ING.UpChan <- true
		}

	} else {
		for _, pp := range ING.Slices {
			uploadTask := task.GetUploadSliceTask(pp, target.FileHash, target.TaskId, target.SpP2PAddress,
				target.IsVideoStream, target.IsEncrypted, ING.FileCRC)
			if uploadTask != nil {
				UploadFileSlice(uploadTask, target.Sign)
			}
		}
		utils.DebugLog("all slices of the task have begun uploading")
		_, ok := <-ING.UpChan
		if ok {
			close(ING.UpChan)
		}
		task.UpIngMap.Delete(target.FileHash)

		if target.IsVideoStream {
			file.DeleteTmpHlsFolder(target.FileHash)
		}
	}
}

func uploadKeep(fileHash, taskID string) {
	utils.DebugLogf("uploadKeep  fileHash = %v  taskID = %v", fileHash, taskID)
	if ing, ok := task.UpIngMap.Load(fileHash); ok {
		ING := ing.(*task.UpFileIng)
		ING.UpChan <- true
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
	task.UpIngMap.Delete(fileHash)
	task.UploadProgressMap.Delete(fileHash)
}
