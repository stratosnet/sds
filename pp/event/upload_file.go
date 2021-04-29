package event

// Author j
import (
	"context"
	"fmt"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/httpserv"
	"net/http"
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
	p := RequestUploadFileData(tmpString, "", reqID, true)
	SendMessageToSPServer(p, header.ReqUploadFile)
	stroeResponseWriter(reqID, w)
}

// RequestUploadFile request to SP for upload file
func RequestUploadFile(path, reqID string, _ http.ResponseWriter) {
	utils.DebugLog("______________path", path)
	if !setting.CheckLogin() {
		return
	}
	isFile, err := file.IsFile(path)
	fileHash := file.GetFileHash(path)
	data := make(map[string]string)
	if err != nil {
		fmt.Println(err)
		return
	}
	if isFile {
		p := RequestUploadFileData(path, "", reqID, false)
		SendMessageToSPServer(p, header.ReqUploadFile)
		data["fileHash"] = fileHash

		return
	}
	// is directory
	utils.DebugLog("this is a directory, not file")
	file.GetAllFiles(path)

	for {
		select {
		case pathString := <-setting.UpChan:
			utils.DebugLog("path string == ", pathString)
			p := RequestUploadFileData(pathString, "", reqID, false)
			SendMessageToSPServer(p, header.ReqUploadFile)
		default:
			return
		}
	}
}

// RspUploadFile response of upload file event
func RspUploadFile(ctx context.Context, _ spbf.WriteCloser) {
	utils.DebugLog("get RspUploadFile")
	target := &protos.RspUploadFile{}
	if !unmarshalData(ctx, target) {
		utils.ErrorLog("unmarshal error")
		return
	}
	// upload file to PP based on the PP info provided by SP
	if target.Result == nil {
		utils.ErrorLog("target.Result is nil")

	} else if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.Log("upload failed", target.Result.Msg)
		file.ClearFileMap(target.FileHash)

	} else if len(target.PpList) != 0 {
		go startUploadTask(target)

	} else {
		fmt.Println("file upload successfully！  target.PpList", target.FileHash)
		var p float32 = 100
		ProgressMap.Store(target.FileHash, p)
		task.UpLoadProgressMap.Delete(target.FileHash)

	}
	if isCover {
		utils.DebugLog("is_cover", target.ReqId)
		putData(target.ReqId, HTTPUploadFile, target)
	}

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
	}
	task.UpIngMap.Store(target.FileHash, taskING)
	if client.PPConn != nil {
		conn := client.NewClient(client.PPConn.GetName(), setting.IsPP)
		client.UpConnMap.Store(target.FileHash, conn)
		utils.DebugLog("task.UpConnMap.Store(target.FileHash, conn)", target.FileHash)
	}
	if prg, ok := task.UpLoadProgressMap.Load(target.FileHash); ok {
		progress := prg.(*task.UpProgress)
		progress.HasUpload = (target.TotalSlice - int64(len(target.PpList))) * 32 * 1024 * 1024
	}
	go sendUploadFileSlice(target.FileHash, target.TaskId)

}

func up(ING *task.UpFileIng, fileHash string) {
	for {
		select {
		case goon := <-ING.UpChan:
			if !goon {
				continue
			}

			if len(ING.Slices) == 0 {
				utils.DebugLog("all slices of the task are uploaded")
				if _, ok := <-ING.UpChan; ok {
					close(ING.UpChan)
				}
				task.UpIngMap.Delete(fileHash)
				return
			}
			pp := ING.Slices[0]
			utils.DebugLog("start upload!!!!!", pp.SliceNumber)
			uploadTask := task.GetUploadSliceTask(pp, ING.FileHash, ING.TaskID)
			if uploadTask == nil {
				continue
			}

			if _, ok := client.UpConnMap.Load(fileHash); !ok {
				return
			}
			UploadFileSlice(uploadTask)
			ING.Slices = append(ING.Slices[:0], ING.Slices[0+1:]...)
		}
	}
}

func sendUploadFileSlice(fileHash, taskID string) {
	ing, ok := task.UpIngMap.Load(fileHash)
	if !ok {
		utils.DebugLog("all slices of the task are uploaded")
		return
	}
	ING := ing.(*task.UpFileIng)
	if len(ING.Slices) > task.MAXSLICE {
		go up(ING, fileHash)
		for i := 0; i < task.MAXSLICE; i++ {
			ING.UpChan <- true
		}

	} else {
		for _, pp := range ING.Slices {
			if _, ok := client.UpConnMap.Load(fileHash); !ok {
				return
			}
			uploadTask := task.GetUploadSliceTask(pp, fileHash, taskID)
			if uploadTask != nil {
				UploadFileSlice(uploadTask)
			}
		}
		utils.DebugLog("all slices of the task are uploaded")
		_, ok := <-ING.UpChan
		if ok {
			close(ING.UpChan)
		}
		task.UpIngMap.Delete(fileHash)
	}
}

func uploadKeep(fileHash, taskID string) {
	utils.DebugLog("uploadKeep", fileHash, taskID)
	if ing, ok := task.UpIngMap.Load(fileHash); ok {
		ING := ing.(*task.UpFileIng)
		ING.UpChan <- true
	}
}

// UploadPause
func UploadPause(fileHash, reqID string, w http.ResponseWriter) {
	if c, ok := client.UpConnMap.Load(fileHash); ok {
		conn := c.(*cf.ClientConn)
		conn.ClientClose()
		utils.DebugLog("UploadPause", conn)
	}
	client.UpConnMap.Delete(fileHash)
	task.UpIngMap.Delete(fileHash)
	task.UpLoadProgressMap.Delete(fileHash)
}
