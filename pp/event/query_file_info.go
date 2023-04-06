package event

// Author j
import (
	"context"
	"io/ioutil"
	"net/http"
	"sync"
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
)

var (
	requestDownloadMap = &sync.Map{}
)

// GetFileStorageInfo p to pp. The downloader is assumed the default wallet of this node, if this function is invoked.
func GetFileStorageInfo(ctx context.Context, path, savePath, saveAs string, isVideoStream bool, w http.ResponseWriter) {
	utils.DebugLog("GetFileStorageInfo")

	if !setting.CheckLogin() {
		notLogin(w)
		return
	}
	if len(path) < setting.Config.DownloadPathMinLen {
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

	req := requests.ReqFileStorageInfoData(path, savePath, saveAs, setting.WalletAddress, setting.WalletPublicKey, isVideoStream, nil)
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
			P2PAddress:    setting.P2PAddress,
		}
		p2pServer := p2pserver.GetP2pServer(ctx)
		_ = p2pServer.SendMessage(ctx, p2pServer.GetPpConn(), req, header.ReqClearDownloadTask)
		_, _ = w.Write([]byte("ok"))
	} else {
		notLogin(w)
	}
}

func ReqClearDownloadTask(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqClearDownloadTask
	if err := VerifyMessage(ctx, header.ReqClearDownloadTask, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
	}
	if requests.UnmarshalData(ctx, &target) {
		task.DeleteDownloadTask(target.WalletAddress, target.WalletAddress, "")
	}
}

func GetVideoSliceInfo(ctx context.Context, sliceName string, fInfo *protos.RspFileStorageInfo) *protos.DownloadSliceInfo {
	var sliceNumber uint64
	hlsInfo := GetHlsInfo(ctx, fInfo)
	sliceNumber = hlsInfo.SegmentToSlice[sliceName]
	sliceInfo := GetSliceInfoBySliceNumber(fInfo, sliceNumber)
	return sliceInfo
}

func GetVideoSlice(ctx context.Context, sliceInfo *protos.DownloadSliceInfo, fInfo *protos.RspFileStorageInfo, w http.ResponseWriter) {
	if !setting.CheckLogin() {
		notLogin(w)
		return
	}

	utils.DebugLog("taskid ======= ", sliceInfo.TaskId)
	sliceHash := sliceInfo.SliceStorageInfo.SliceHash
	if file.CheckSliceExisting(fInfo.FileHash, fInfo.FileName, sliceHash, fInfo.SavePath, fInfo.ReqId) {
		utils.Log("slice exist already,", sliceHash)
		slicePath := file.GetDownloadTmpPath(fInfo.FileHash, sliceHash, fInfo.SavePath)
		video, _ := ioutil.ReadFile(slicePath)
		_, _ = w.Write(video)
	} else {
		req := requests.ReqDownloadSliceData(fInfo, sliceInfo)
		newCtx := createAndRegisterSliceReqId(ctx, fInfo.ReqId)
		utils.Log("Send request for downloading slice: ", sliceInfo.SliceStorageInfo.SliceHash)
		SendReqDownloadSlice(newCtx, fInfo.FileHash, sliceInfo, req, fInfo.ReqId)
		if err := storeResponseWriter(newCtx, w); err != nil {
			w.WriteHeader(setting.FAILCode)
			_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "Get video segment time out").ToBytes())
		}
	}
}

func GetVideoSlices(ctx context.Context, fInfo *protos.RspFileStorageInfo, dTask *task.DownloadTask) {
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
		go cacheSlice(ctx, videoCacheTask, fInfo, dTask)
		for i := 0; i < setting.STREAM_CACHE_MAXSLICE; i++ {
			videoCacheTask.DownloadCh <- true
		}
	} else {
		for _, sliceInfo := range videoCacheTask.Slices {
			if !file.CheckSliceExisting(fInfo.FileHash, fInfo.FileName, sliceInfo.SliceStorageInfo.SliceHash, fInfo.SavePath, fInfo.ReqId) {

				req := requests.ReqDownloadSliceData(fInfo, sliceInfo)
				newCtx := createAndRegisterSliceReqId(ctx, fInfo.ReqId)
				req.IsVideoCaching = true
				SendReqDownloadSlice(newCtx, fInfo.FileHash, sliceInfo, req, fInfo.ReqId)
			} else {
				task.CleanDownloadTask(ctx, fInfo.FileHash, sliceInfo.SliceStorageInfo.SliceHash, setting.WalletAddress, task.LOCAL_REQID)
				setDownloadSliceSuccess(ctx, sliceInfo.SliceStorageInfo.SliceHash, dTask)
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

func cacheSlice(ctx context.Context, videoCacheTask *task.VideoCacheTask, fInfo *protos.RspFileStorageInfo, dTask *task.DownloadTask) {
	for goon := range videoCacheTask.DownloadCh {
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
		if file.CheckSliceExisting(fInfo.FileHash, fInfo.FileName, sliceInfo.SliceStorageInfo.SliceHash, fInfo.SavePath, fInfo.ReqId) {
			utils.DebugLog("slice exist already ", sliceInfo.SliceNumber)
			task.CleanDownloadTask(ctx, fInfo.FileHash, sliceInfo.SliceStorageInfo.SliceHash, setting.WalletAddress, task.LOCAL_REQID)
			setDownloadSliceSuccess(ctx, sliceInfo.SliceStorageInfo.SliceHash, dTask)
			videoCacheTask.DownloadCh <- true
		} else {
			req := requests.ReqDownloadSliceData(fInfo, sliceInfo)
			newCtx := createAndRegisterSliceReqId(ctx, fInfo.ReqId)
			req.IsVideoCaching = true
			SendReqDownloadSlice(newCtx, fInfo.FileHash, sliceInfo, req, fInfo.ReqId)
		}

		videoCacheTask.Slices = append(videoCacheTask.Slices[:0], videoCacheTask.Slices[0+1:]...)
	}
}

func GetHlsInfo(ctx context.Context, fInfo *protos.RspFileStorageInfo) *file.HlsInfo {
	sliceInfo := GetSliceInfoBySliceNumber(fInfo, uint64(1))
	sliceHash := sliceInfo.SliceStorageInfo.SliceHash
	if !file.CheckSliceExisting(fInfo.FileHash, fInfo.FileName, sliceHash, fInfo.SavePath, fInfo.ReqId) {
		req := requests.ReqDownloadSliceData(fInfo, sliceInfo)
		newCtx := createAndRegisterSliceReqId(ctx, fInfo.ReqId)
		SendReqDownloadSlice(newCtx, fInfo.FileHash, sliceInfo, req, fInfo.ReqId)

		start := time.Now().Unix()
		for {
			if file.CheckSliceExisting(fInfo.FileHash, fInfo.FileName, sliceHash, fInfo.SavePath, fInfo.ReqId) {
				return file.LoadHlsInfo(fInfo.FileHash, sliceHash, fInfo.SavePath)
			} else {
				time.Sleep(time.Second)
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
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	// SPAM check
	if time.Now().Unix()-target.TimeStamp > setting.SPAM_THRESHOLD_SP_SIGN_LATENCY {
		pp.ErrorLog(ctx, "sp's upload file response was expired")
		return
	}

	if target.Result.State == protos.ResultState_RES_FAIL {
		pp.ErrorLog(ctx, "Received fail massage from sp: ", target.Result.Msg)
		return
	}
	metrics.DownloadPerformanceLogNow(target.FileHash + ":RCV_STORAGE_INFO_SP:")

	// check if this is the response from an earlier request of mine
	if fh, found := requestDownloadMap.LoadAndDelete(requests.GetReqIdFromMessage(ctx)); !found || target.FileHash != fh {
		pp.ErrorLog(ctx, "file download response doesn't match the request")
		return
	}

	var newTarget protos.RspFileStorageInfo
	newTarget = target
	fileReqId := core.GetRemoteReqId(ctx)
	newTarget.ReqId = fileReqId
	pp.DebugLog(ctx, "file hash, reqid:", target.FileHash, fileReqId)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		task.CleanDownloadFileAndConnMap(ctx, target.FileHash, fileReqId)
		task.DownloadFileMap.Store(target.FileHash+fileReqId, &newTarget)
		task.AddDownloadTask(&newTarget)
		if target.IsVideoStream {
			return
		}
		DownloadFileSlice(ctx, &target, fileReqId)
	} else {
		file.SetRemoteFileResult(target.FileHash+fileReqId, rpc.Result{Return: rpc.FILE_REQ_FAILURE})
		pp.Log(ctx, "failed to download，", target.Result.Msg)
	}
}

func CheckDownloadPath(path string) bool {
	_, _, _, _, err := datamesh.ParseFileHandle(path)
	return err == nil
}

func StoreDownloadReqId(ctx context.Context, fileHash string) context.Context {
	ctx = core.CreateContextWithReqId(ctx, requests.GetReqIdFromMessage(ctx))
	requestDownloadMap.Store(core.GetReqIdFromContext(ctx), fileHash)
	return ctx
}
