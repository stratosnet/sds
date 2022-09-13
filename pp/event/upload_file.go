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
	"github.com/stratosnet/sds/utils/types"
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
	p := requests.RequestUploadFileData(ctx, tmpString, "", reqID, true, false, false)
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
		p := requests.RequestUploadFileData(ctx, path, "", reqID, false, false, isEncrypted)
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
			p := requests.RequestUploadFileData(ctx, pathString, "", reqID, false, false, isEncrypted)
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
		p := requests.RequestUploadFileData(ctx, path, "", reqID, false, true, false)
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
		FileHash: fileHash,
		Address:  setting.GetPPInfo(),
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

	// verify if wallet matches public key
	if types.VerifyWalletAddrBytes(target.WalletPubkey, target.OwnerWalletAddress) != 0 {
		pp.ErrorLog(ctx, "wallet key validation failed", target.WalletPubkey, target.OwnerWalletAddress)
		return
	}

	// verify wallet signature
	if !types.VerifyWalletSignBytes(target.WalletPubkey, target.WalletSign, utils.GetFileUploadWalletSignMessage(target.FileHash, target.OwnerWalletAddress)) {
		pp.ErrorLog(ctx, "wallet sign failed")
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
	msg := utils.GetRspUploadFileSpNodeSignMessage(setting.P2PAddress, target.SpP2PAddress, target.FileHash, header.RspUploadFile)
	if !types.VerifyP2pSignBytes(spP2pPubkey, target.NodeSign, msg) {
		pp.ErrorLog(ctx, "failed verifying sp's node signature")
		return
	}

	if len(target.PpList) != 0 {
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
	for _, slice := range target.PpList {
		totalSize += int64(slice.GetSliceSize())
	}
	uploadTask := task.CreateUploadFileTask(target.FileHash, target.TaskId, target.SpP2PAddress, false, false, target.Sign, target.PpList, protos.UploadType_BACKUP)
	task.CleanUpConnMap(target.FileHash)
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
	var slices []*protos.SliceHashAddr
	for _, slice := range target.PpList {
		slices = append(slices, &protos.SliceHashAddr{
			SliceHash:   "",
			SliceNumber: slice.SliceNumber,
			SliceSize:   slice.SliceOffset.SliceOffsetEnd - slice.SliceOffset.SliceOffsetStart,
			SliceOffset: slice.SliceOffset,
			PpInfo:      slice.PpInfo,
			SpNodeSign:  slice.SpNodeSign,
		})
	}

	// Create upload task
	uploadTask := task.CreateUploadFileTask(target.FileHash, target.TaskId, target.SpP2PAddress, target.IsEncrypted, target.IsVideoStream, target.NodeSign, slices, protos.UploadType_NEW_UPLOAD)

	task.CleanUpConnMap(target.FileHash)
	task.UploadFileTaskMap.Store(target.FileHash, uploadTask)

	// Video Hls transformation and upload progress update
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
		progress := prg.(*task.UploadProgress)
		if target.IsVideoStream {
			jsonStr, _ := json.Marshal(hlsInfo)
			progress.Total = streamTotalSize + int64(len(jsonStr))
		}
		progress.HasUpload = (target.TotalSlice - int64(len(target.PpList))) * 32 * 1024 * 1024
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

	if fileTask.IsVideoStream {
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
			peers.SendMessageToSPServer(ctx, requests.ReqUploadSlicesWrong(uploadTask, uploadTask.SpP2pAddress, slicesToReDownload, failedSlices), header.ReqUploadSlicesWrong)
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
		case protos.UploadType_BACKUP:
			uploadSliceTask, err = task.GetReuploadSliceTask(ctx, slice, destination.PpInfo, uploadTask)
		}
		if err != nil {
			slice.SetError(err, true, uploadTask)
			return
		}

		pp.DebugLogf(ctx, "starting to upload slice %v for file %v", slice.SliceNumber, uploadTask.FileHash)
		err = UploadFileSlice(ctx, uploadSliceTask)
		if err != nil {
			slice.SetError(err, false, uploadTask)
			return
		}
		slice.SetStatus(task.SLICE_STATUS_FINISHED, uploadTask)
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
	task.UploadFileTaskMap.Delete(fileHash)
	task.UploadProgressMap.Delete(fileHash)
}
