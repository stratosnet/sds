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
	"github.com/stratosnet/sds/pp"
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
	ctx := context.Background()
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
	p := requests.RequestUploadFileData(ctx, tmpString, "", reqID, setting.WalletAddress, true, false, false)
	peers.SendMessageToSPServer(ctx, p, header.ReqUploadFile)
	storeResponseWriter(reqID, w)
}

// RequestUploadFile request to SP for upload file
func RequestUploadFile(ctx context.Context, path, reqID string, isEncrypted bool, _ http.ResponseWriter) {
	pp.DebugLog(ctx, "______________path", path)
	if !setting.CheckLogin() {
		return
	}

	isFile, err := file.IsFile(path)
	if err != nil {
		utils.ErrorLog(err)
		return
	}
	if isFile {
		p := requests.RequestUploadFileData(ctx, path, "", reqID, setting.WalletAddress, false, false, isEncrypted)
		peers.SendMessageToSPServer(ctx, p, header.ReqUploadFile)
		return
	}

	// is directory
	pp.DebugLog(ctx, "this is a directory, not file")
	file.GetAllFiles(path)
	for {
		select {
		case pathString := <-setting.UpChan:
			pp.DebugLog(ctx, "path string == ", pathString)
			p := requests.RequestUploadFileData(ctx, pathString, "", reqID, setting.WalletAddress, false, false, isEncrypted)
			peers.SendMessageToSPServer(ctx, p, header.ReqUploadFile)
		default:
			return
		}
	}
}

func RequestUploadStream(ctx context.Context, path, reqID string, _ http.ResponseWriter) {
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
		p := requests.RequestUploadFileData(ctx, path, "", reqID, setting.WalletAddress, false, true, false)
		if p != nil {
			peers.SendMessageToSPServer(ctx, p, header.ReqUploadFile)
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
		FileHash:      fileHash,
		P2PAddress:    setting.P2PAddress,
		WalletAddress: setting.WalletAddress,
	}
	peers.SendMessageToSPServer(ctx, p, header.ReqFileBackupStatus)
}

// RspUploadFile response of upload file event
func RspUploadFile(ctx context.Context, _ core.WriteCloser) {
	pp.DebugLog(ctx, "get RspUploadFile")
	target := &protos.RspUploadFile{}
	if !requests.UnmarshalData(ctx, target) {
		pp.ErrorLog(ctx, "unmarshal error")
		return
	}
	// upload file to PP based on the PP info provided by SP
	if target.Result == nil {
		pp.ErrorLog(ctx, "target.Result is nil")

	} else if target.Result.State != protos.ResultState_RES_SUCCESS {
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

	} else if len(target.PpList) != 0 {
		go startUploadTask(ctx, target)

	} else {
		pp.Log(ctx, "file upload successful！  fileHash", target.FileHash)
		var p float32 = 100
		ProgressMap.Store(target.FileHash, p)
		task.UploadProgressMap.Delete(target.FileHash)

		if file.IsFileRpcRemote(target.FileHash) {
			file.SetRemoteFileResult(target.FileHash, rpc.Result{Return: rpc.SUCCESS})
		}
	}
	if isCover {
		pp.DebugLog(ctx, "is_cover", target.ReqId)
		putData(target.ReqId, HTTPUploadFile, target)
	}

}

func RspBackupStatus(ctx context.Context, _ core.WriteCloser) {
	pp.DebugLog(ctx, "get RspBackupStatus")
	target := &protos.RspBackupStatus{}
	if !requests.UnmarshalData(ctx, target) {
		pp.ErrorLog(ctx, "unmarshal error")
		return
	}

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

	if len(target.PpList) == 0 {
		ScheduleReqBackupStatus(ctx, target.FileHash)
		return
	}

	pp.Logf(ctx, "Start re-uploading slices for the file  %s", target.FileHash)
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
			UploadFileSlice(ctx, uploadTask, target.Sign)
		}
	}
	pp.DebugLog(ctx, "all slices of the task have begun uploading")
	close(taskING.UpChan)
	task.UpIngMap.Delete(target.FileHash)
}

// startUploadTask
func startUploadTask(ctx context.Context, target *protos.RspUploadFile) {
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
		file.VideoToHls(ctx, target.FileHash)
		if hlsInfo, err := file.GetHlsInfo(target.FileHash, uint64(len(target.PpList))); err != nil {
			pp.ErrorLog(ctx, "Hls transformation failed: ", err)
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
	go sendUploadFileSlice(ctx, target)
}

func up(ctx context.Context, ING *task.UpFileIng, target *protos.RspUploadFile) {
	for {
		select {
		case goon := <-ING.UpChan:
			if !goon {
				continue
			}

			if len(ING.Slices) == 0 {
				pp.DebugLog(ctx, "all slices of the task have begun uploading")
				if _, ok := <-ING.UpChan; ok {
					close(ING.UpChan)
				}
				task.UpIngMap.Delete(target.FileHash)

				if target.IsVideoStream {
					file.DeleteTmpHlsFolder(ctx, target.FileHash)
				}

				return
			}
			ppNode := ING.Slices[0]
			pp.DebugLog(ctx, "start upload!!!!!", ppNode.SliceNumber)
			uploadTask := task.GetUploadSliceTask(ctx, ppNode, ING.FileHash, ING.TaskID, target.SpP2PAddress,
				target.IsVideoStream, target.IsEncrypted, ING.FileCRC)
			if uploadTask == nil {
				continue
			}

			UploadFileSlice(ctx, uploadTask, target.Sign)
			ING.Slices = append(ING.Slices[:0], ING.Slices[0+1:]...)
		}
	}
}

func sendUploadFileSlice(ctx context.Context, target *protos.RspUploadFile) {
	ing, ok := task.UpIngMap.Load(target.FileHash)
	if !ok {
		pp.DebugLog(ctx, "all slices of the task have begun uploading")
		return
	}
	ING := ing.(*task.UpFileIng)
	if len(ING.Slices) > task.MAXSLICE {
		go up(ctx, ING, target)
		for i := 0; i < task.MAXSLICE; i++ {
			ING.UpChan <- true
		}

	} else {
		for _, ppNode := range ING.Slices {
			uploadTask := task.GetUploadSliceTask(ctx, ppNode, target.FileHash, target.TaskId, target.SpP2PAddress,
				target.IsVideoStream, target.IsEncrypted, ING.FileCRC)
			if uploadTask != nil {
				UploadFileSlice(ctx, uploadTask, target.Sign)
			}
		}
		pp.DebugLog(ctx, "all slices of the task have begun uploading")
		_, ok := <-ING.UpChan
		if ok {
			close(ING.UpChan)
		}
		task.UpIngMap.Delete(target.FileHash)

		if target.IsVideoStream {
			file.DeleteTmpHlsFolder(ctx, target.FileHash)
		}
	}
}

func uploadKeep(ctx context.Context, fileHash, taskID string) {
	pp.DebugLogf(ctx, "uploadKeep  fileHash = %v  taskID = %v", fileHash, taskID)
	if ing, ok := task.UpIngMap.Load(fileHash); ok {
		ING := ing.(*task.UpFileIng)
		ING.UpChan <- true
	}
}

// UploadPause
func UploadPause(ctx context.Context, fileHash, reqID string, w http.ResponseWriter) {
	client.UpConnMap.Range(func(k, v interface{}) bool {
		if strings.HasPrefix(k.(string), fileHash) {
			conn := v.(*cf.ClientConn)
			conn.ClientClose()
			pp.DebugLog(ctx, "UploadPause", conn)
		}
		return true
	})
	task.CleanUpConnMap(fileHash)
	task.UpIngMap.Delete(fileHash)
	task.UploadProgressMap.Delete(fileHash)
}
