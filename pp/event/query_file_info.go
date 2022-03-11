package event

// Author j
import (
	"context"
	"io/ioutil"
	"net/http"
	"time"

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
	"github.com/stratosnet/sds/utils/datamesh"
	"github.com/stratosnet/sds/utils/httpserv"
)

// GetFileStorageInfo p to pp
func GetFileStorageInfo(path, savePath, reqID, walletAddr string, isVideoStream bool, w http.ResponseWriter) {
	if setting.CheckLogin() {
		if CheckDownloadPath(path) {
			utils.DebugLog("path:", path)
			req := requests.ReqFileStorageInfoData(path, savePath, reqID, walletAddr, isVideoStream, nil)
			peers.SendMessageDirectToSPOrViaPP(req, header.ReqFileStorageInfo)
		} else {
			utils.ErrorLog("please input correct download link, eg: sdm://address/fileHash|filename(optional)")
			if w != nil {
				w.Write(httpserv.NewJson(nil, setting.FAILCode, "please input correct download link, eg:  sdm://address/fileHash|filename(optional)").ToBytes())
			}
		}
	} else {
		notLogin(w)
	}
}

func ClearFileInfoAndDownloadTask(fileHash string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		task.DownloadFileMap.Delete(fileHash)
		task.DeleteDownloadTask(fileHash, setting.WalletAddress)
		req := &protos.ReqClearDownloadTask{
			WalletAddress: setting.WalletAddress,
			FileHash:      fileHash,
			P2PAddress:    setting.P2PAddress,
		}
		peers.SendMessage(client.PPConn, req, header.ReqClearDownloadTask)
		w.Write([]byte("ok"))
	} else {
		notLogin(w)
	}
}

func ReqClearDownloadTask(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqClearDownloadTask
	if requests.UnmarshalData(ctx, &target) {
		task.DeleteDownloadTask(target.WalletAddress, target.WalletAddress)
	}
}

func GetVideoSliceInfo(sliceName string, fInfo *protos.RspFileStorageInfo) *protos.DownloadSliceInfo {
	var sliceNumber uint64
	hlsInfo := GetHlsInfo(fInfo)
	sliceNumber = hlsInfo.SegmentToSlice[sliceName]
	sliceInfo := GetSliceInfoBySliceNumber(fInfo, sliceNumber)
	return sliceInfo
}

func GetVideoSlice(sliceInfo *protos.DownloadSliceInfo, fInfo *protos.RspFileStorageInfo, w http.ResponseWriter) {
	if setting.CheckLogin() {
		utils.DebugLog("taskid ======= ", sliceInfo.TaskId)
		sliceHash := sliceInfo.SliceStorageInfo.SliceHash
		if file.CheckSliceExisting(fInfo.FileHash, fInfo.FileName, sliceHash, fInfo.SavePath) {
			utils.Log("slice exist already,", sliceHash)
			slicePath := file.GetDownloadTmpPath(fInfo.FileHash, sliceHash, fInfo.SavePath)
			video, _ := ioutil.ReadFile(slicePath)
			w.Write(video)
		} else {
			req := requests.ReqDownloadSliceData(fInfo, sliceInfo)
			utils.Log("Send request for downloading slice: ", sliceInfo.SliceStorageInfo.SliceHash)
			SendReqDownloadSlice(fInfo.FileHash, sliceInfo, req)
			if err := storeResponseWriter(req.ReqId, w); err != nil {
				w.WriteHeader(setting.FAILCode)
				w.Write(httpserv.NewErrorJson(setting.FAILCode, "Get video segment time out").ToBytes())
			}
		}
	} else {
		notLogin(w)
	}
}

func GetVideoSlices(fInfo *protos.RspFileStorageInfo) {
	slices := make([]*protos.DownloadSliceInfo, len(fInfo.SliceInfo))

	// reverse order to download start from last slice
	for i := 0; i < len(fInfo.SliceInfo); i++ {
		idx := uint64(len(fInfo.SliceInfo)) - fInfo.SliceInfo[i].SliceNumber
		slices[idx] = fInfo.SliceInfo[i]
	}

	videoCacheTask := &task.VideoCacheTask{
		Slices:     slices,
		FileHash:   fInfo.FileHash,
		DownloadCh: make(chan bool, setting.STREAM_CACHE_MAXSLICE),
	}

	task.VideoCacheTaskMap.Store(fInfo.FileHash, videoCacheTask)

	if len(videoCacheTask.Slices) > setting.STREAM_CACHE_MAXSLICE {
		go cacheSlice(videoCacheTask, fInfo)
		for i := 0; i < setting.STREAM_CACHE_MAXSLICE; i++ {
			videoCacheTask.DownloadCh <- true
		}
	} else {
		for _, sliceInfo := range videoCacheTask.Slices {
			if !file.CheckSliceExisting(fInfo.FileHash, fInfo.FileName, sliceInfo.SliceStorageInfo.SliceHash, fInfo.SavePath) {

				req := requests.ReqDownloadSliceData(fInfo, sliceInfo)
				req.IsVideoCaching = true
				if req != nil {
					SendReqDownloadSlice(fInfo.FileHash, sliceInfo, req)
				}
			}
		}
		utils.DebugLog("all slices of the task have begun downloading")
		_, ok := <-videoCacheTask.DownloadCh
		if ok {
			close(videoCacheTask.DownloadCh)
		}
		task.VideoCacheTaskMap.Delete(fInfo.FileHash)
	}
}

func cacheSlice(videoCacheTask *task.VideoCacheTask, fInfo *protos.RspFileStorageInfo) {
	for {
		select {
		case goon := <-videoCacheTask.DownloadCh:
			if !goon {
				continue
			}

			if len(videoCacheTask.Slices) == 0 {
				utils.DebugLog("all slices of the task have begun downloading")
				if _, ok := <-videoCacheTask.DownloadCh; ok {
					close(videoCacheTask.DownloadCh)
				}
				task.VideoCacheTaskMap.Delete(videoCacheTask.FileHash)
				return
			}
			sliceInfo := videoCacheTask.Slices[0]
			utils.DebugLog("start Download!!!!!", sliceInfo.SliceNumber)
			if file.CheckSliceExisting(fInfo.FileHash, fInfo.FileName, sliceInfo.SliceStorageInfo.SliceHash, fInfo.SavePath) {
				utils.DebugLog("slice exist already ", sliceInfo.SliceNumber)
				videoCacheTask.DownloadCh <- true
			} else {
				req := requests.ReqDownloadSliceData(fInfo, sliceInfo)
				req.IsVideoCaching = true
				SendReqDownloadSlice(fInfo.FileHash, sliceInfo, req)
			}

			videoCacheTask.Slices = append(videoCacheTask.Slices[:0], videoCacheTask.Slices[0+1:]...)
		}
	}
}

func GetHlsInfo(fInfo *protos.RspFileStorageInfo) *file.HlsInfo {
	sliceInfo := GetSliceInfoBySliceNumber(fInfo, uint64(1))
	sliceHash := sliceInfo.SliceStorageInfo.SliceHash
	if !file.CheckSliceExisting(fInfo.FileHash, fInfo.FileName, sliceHash, fInfo.SavePath) {
		req := requests.ReqDownloadSliceData(fInfo, sliceInfo)
		SendReqDownloadSlice(fInfo.FileHash, sliceInfo, req)

		start := time.Now().Unix()
		for {
			if file.CheckSliceExisting(fInfo.FileHash, fInfo.FileName, sliceHash, fInfo.SavePath) {
				return file.LoadHlsInfo(fInfo.FileHash, sliceHash, fInfo.SavePath)
			} else {
				select {
				case <-time.After(time.Second):
				}
				if time.Now().Unix()-start > setting.HTTPTIMEOUT {
					return nil
				}
			}
		}
	} else {
		return file.LoadHlsInfo(fInfo.FileHash, sliceHash, fInfo.SavePath)
	}
}

func GetSliceInfoBySliceNumber(fInfo *protos.RspFileStorageInfo, sliceNumber uint64) *protos.DownloadSliceInfo {
	for _, slice := range fInfo.SliceInfo {
		if slice.SliceNumber == sliceNumber {
			return slice
		}
	}
	return nil
}

// ReqFileStorageInfo  P-PP , PP-SP
func ReqFileStorageInfo(ctx context.Context, conn core.WriteCloser) {
	utils.Log("pp get ReqFileStorageInfo directly transfer to SP")
	peers.TransferSendMessageToSPServer(core.MessageFromContext(ctx))
}

// RspFileStorageInfo SP-PP , PP-P
func RspFileStorageInfo(ctx context.Context, conn core.WriteCloser) {
	// PP check whether itself is the storage PP, if not transfer
	utils.Log("get，RspFileStorageInfo")
	var target protos.RspFileStorageInfo
	if requests.UnmarshalData(ctx, &target) {
		utils.DebugLog("file hash", target.FileHash)
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			utils.Log("download starts: ")
			task.CleanDownloadFileAndConnMap(target.FileHash)
			task.DownloadFileMap.Store(target.FileHash, &target)
			task.AddDownloadTask(&target)
			if target.IsVideoStream {
				return
			}
			DownloadFileSlice(&target)
			utils.DebugLog("DownloadFileSlice(&target)", target)
		} else {
			utils.Log("failed to download，", target.Result.Msg)
		}
	}
}

// CheckDownloadPath
func CheckDownloadPath(path string) bool {

	if len(path) < setting.Config.DownloadPathMinLen {
		utils.DebugLog("invalid path length")
		return false
	}
	if path[:6] != datamesh.DATA_MASH_PREFIX {
		return false
	}
	if path[47:48] != "/" {
		return false
	}
	return true
}
