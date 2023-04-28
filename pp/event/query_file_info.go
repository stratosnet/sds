package event

// Author j
import (
	"context"
	"net/http"
	"os"
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
	"github.com/stratosnet/sds/utils/types"
)

// GetFileStorageInfo p to pp. The downloader is assumed the default wallet of this node, if this function is invoked.
func GetFileStorageInfo(ctx context.Context, path, savePath, saveAs string, w http.ResponseWriter) {
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

	req := requests.ReqFileStorageInfoData(ctx, path, savePath, saveAs, setting.WalletAddress, setting.WalletPublicKey, nil)
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
			P2PAddress:    p2pserver.GetP2pServer(ctx).GetP2PAddress(),
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
		video, _ := os.ReadFile(slicePath)
		_, _ = w.Write(video)
	} else {
		req := requests.ReqDownloadSliceData(ctx, fInfo, sliceInfo)
		newCtx := createAndRegisterSliceReqId(ctx, fInfo.ReqId)
		utils.Log("Send request for downloading slice: ", sliceInfo.SliceStorageInfo.SliceHash)
		SendReqDownloadSlice(newCtx, fInfo.FileHash, sliceInfo, req, fInfo.ReqId)
		if err := storeResponseWriter(newCtx, w); err != nil {
			w.WriteHeader(setting.FAILCode)
			_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "Get video segment time out").ToBytes())
		}
	}
}

func GetVideoSlices(ctx context.Context, fInfo *protos.RspFileStorageInfo, fileReqId string) {
	dTask, _ := task.GetDownloadTask(fInfo.FileHash, setting.WalletAddress, fileReqId)
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

				req := requests.ReqDownloadSliceData(ctx, fInfo, sliceInfo)
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
			req := requests.ReqDownloadSliceData(ctx, fInfo, sliceInfo)
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
		req := requests.ReqDownloadSliceData(ctx, fInfo, sliceInfo)
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

	savePath := target.SavePath
	isVideoStream := utils.IsVideoStream(target.FileHash)
	if savePath == "" && isVideoStream {
		savePath = setting.VIDEOPATH
	}

	newTarget := &protos.RspFileStorageInfo{
		VisitCer:      target.VisitCer,
		P2PAddress:    target.P2PAddress,
		WalletAddress: target.WalletAddress,
		SliceInfo:     target.SliceInfo,
		FileHash:      target.FileHash,
		FileName:      target.FileName,
		Result:        target.Result,
		ReqId:         target.ReqId,
		SavePath:      target.SavePath,
		FileSize:      target.FileSize,
		RestAddress:   target.RestAddress,
		NodeSign:      target.NodeSign,
		SpP2PAddress:  target.SpP2PAddress,
		EncryptionTag: target.EncryptionTag,
		TaskId:        target.TaskId,
		TimeStamp:     target.TimeStamp,
	}

	fileReqId := core.GetRemoteReqId(ctx)
	newTarget.ReqId = fileReqId
	pp.DebugLog(ctx, "file hash, reqid:", target.FileHash, fileReqId)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		task.CleanDownloadFileAndConnMap(ctx, target.FileHash, fileReqId)
		task.DownloadFileMap.Store(target.FileHash+fileReqId, newTarget)
		task.AddDownloadTask(newTarget)
		if isVideoStream {
			if strCh, ok := task.VideoCacheChannelMap.Load(fileReqId); ok {
				strCh.(chan string) <- target.FileHash
			}
			GetVideoSlices(ctx, newTarget, fileReqId)
		} else {
			DownloadFileSlice(ctx, newTarget, fileReqId)
		}
	} else {
		file.SetRemoteFileResult(target.FileHash+fileReqId, rpc.Result{Return: rpc.FILE_REQ_FAILURE})
		pp.Log(ctx, "failed to download，", target.Result.Msg)
	}
}

func GetFileReplicaInfo(ctx context.Context, path string, replicaIncreaseNum uint32) {
	utils.DebugLog("GetFileReplicaInfo")

	if !setting.CheckLogin() {
		return
	}
	if len(path) < setting.Config.DownloadPathMinLen {
		utils.DebugLog("invalid path length")
		return
	}
	_, _, fileHash, _, err := datamesh.ParseFileHandle(path)
	if err != nil {
		pp.ErrorLog(ctx, "please input correct file link, eg: sdm://address/fileHash|filename(optional)")
		return
	}
	pp.DebugLog(ctx, "path:", path)

	// sign the wallet signature by wallet private key
	wsignMsg := utils.GetFileReplicaInfoWalletSignMessage(fileHash, setting.WalletAddress)
	wsign, err := types.BytesToAccPriveKey(setting.WalletPrivateKey).Sign([]byte(wsignMsg))
	if err != nil {
		return
	}

	req := requests.ReqFileReplicaInfo(path, setting.WalletAddress, p2pserver.GetP2pServer(ctx).GetP2PAddress(), replicaIncreaseNum, wsign, setting.WalletPublicKey)
	p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, req, header.ReqFileReplicaInfo)
}

func RspFileReplicaInfo(ctx context.Context, conn core.WriteCloser) {
	pp.Log(ctx, "get，RspGetFileReplicaInfo")
	var target protos.RspFileReplicaInfo
	if err := VerifyMessage(ctx, header.RspFileReplicaInfo, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
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

func CheckDownloadPath(path string) bool {
	_, _, _, _, err := datamesh.ParseFileHandle(path)
	return err == nil
}
