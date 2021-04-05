package event

// Author j
import (
	"context"
	"fmt"
	"github.com/qsnetwork/sds/framework/client/cf"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg/header"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/pp/client"
	"github.com/qsnetwork/sds/pp/file"
	"github.com/qsnetwork/sds/pp/setting"
	"github.com/qsnetwork/sds/pp/task"
	"github.com/qsnetwork/sds/utils"
	"github.com/qsnetwork/sds/utils/httpserv"
	"net/http"
	"sync"
)

var m *sync.WaitGroup
var is_cover bool

// RequestUploadCoverImage RequestUploadCoverImage
func RequestUploadCoverImage(pathStr, reqID string, w http.ResponseWriter) {
	is_cover = true
	tmpString, err := utils.ImageCommpress(pathStr)
	utils.DebugLog("reqID", reqID)
	if utils.CheckError(err) {
		utils.ErrorLog(err)
		if w != nil {
			w.Write(httpserv.NewJson(nil, setting.FAILCode, "compress image failed").ToBytes())
		}
	} else {
		p := RequestUploadFileData(tmpString, "", reqID, true)
		SendMessageToSPServer(p, header.ReqUploadFile)
		stroeResponseWriter(reqID, w)
	}
}

// RequestUploadFile request to SP for upload file
func RequestUploadFile(path, reqID string, w http.ResponseWriter) {
	utils.DebugLog("______________path", path)
	if setting.CheckLogin() {
		pathType := file.IsFile(path)
		fileHash := file.GetFileHash(path)
		data := make(map[string]string)
		if pathType == 0 {
			fmt.Println("wrong path ")
		} else if pathType == 1 {
			p := RequestUploadFileData(path, "", reqID, false)
			SendMessageToSPServer(p, header.ReqUploadFile)
			data["fileHash"] = fileHash
		} else {
			utils.DebugLog("this is a directory, not file")
			file.GetAllFile(path)

			for {
				select {
				case pathstring := <-setting.UpChan:
					utils.DebugLog("pathstring == ", pathstring)
					p := RequestUploadFileData(pathstring, "", reqID, false)
					SendMessageToSPServer(p, header.ReqUploadFile)
				default:
					return
				}

			}
		}
	}
}

// RspUploadFile
func RspUploadFile(ctx context.Context, conn spbf.WriteCloser) {
	utils.DebugLog("get RspUploadFile")
	var target protos.RspUploadFile
	if unmarshalData(ctx, &target) {
		// upload file to PP based on the PP info provided by SP
		if target.Result != nil {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				if len(target.PpList) == 0 {
					fmt.Println("file upload successfully！  target.PpList", target.FileHash)
					var p float32 = 100
					ProgressMap.Store(target.FileHash, p)
					task.UpLoadProgressMap.Delete(target.FileHash)
				} else {
					go StartUploadTask(&target)
				}

			} else {
				utils.Log("upload failed", target.Result.Msg)
				file.ClearFileMap(target.FileHash)
			}
		} else {
			utils.ErrorLog("target.Result is nil")
		}
		if is_cover {
			utils.DebugLog("is_cover", target.ReqId)
			putData(target.ReqId, HTTPUploadFile, &target)
		}
	} else {
		utils.ErrorLog("unmarshal error")
	}
}

// StartUploadTask
func StartUploadTask(target *protos.RspUploadFile) {
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
			if goon {
				if len(ING.Slices) != 0 {
					pp := ING.Slices[0]
					utils.DebugLog("start upload!!!!!", pp.SliceNumber)
					uploadTask := task.GetUploadSliceTask(pp, ING.FileHash, ING.TaskID)
					if uploadTask != nil {
						if _, ok := client.UpConnMap.Load(fileHash); !ok {
							return
						}
						UploadFileSlice(uploadTask)
						ING.Slices = append(ING.Slices[:0], ING.Slices[0+1:]...)
					}
				} else {
					utils.DebugLog("all slices of the task are uploaded")
					_, ok := <-ING.UpChan
					if ok {
						close(ING.UpChan)
					}
					task.UpIngMap.Delete(fileHash)
					return
				}
			}
		}
	}
}

func sendUploadFileSlice(fileHash, taskID string) {
	if ing, ok := task.UpIngMap.Load(fileHash); ok {
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

	} else {
		utils.DebugLog("all slices of the task are uploaded")
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
