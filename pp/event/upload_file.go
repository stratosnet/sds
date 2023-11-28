package event

// Author j
import (
	"context"
	"encoding/json"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/metrics"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/framework/utils/encryption"
	"github.com/stratosnet/sds/framework/utils/encryption/hdkey"
	"github.com/stratosnet/sds/framework/utils/types"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/sds-msg/header"
	"github.com/stratosnet/sds/sds-msg/protos"
)

var (
	isCover bool
)

// RequestUploadFile request to SP for upload file
func RequestUploadFile(ctx context.Context, path string, isEncrypted, isVideoStream bool, desiredTier uint32, allowHigherTier bool,
	walletAddr string, walletPubkey, wsign []byte) {
	pp.DebugLog(ctx, "______________path", path)
	if !setting.CheckLogin() {
		return
	}

	var fileHash string
	if isVideoStream {
		fileHash = file.GetFileHashForVideoStream(path, "")
	} else {
		fileHash = file.GetFileHash(path, "")
	}

	metrics.UploadPerformanceLogNow(fileHash + ":RCV_CMD_START:")

	isFile, err := file.IsFile(path)
	if err != nil {
		pp.ErrorLog(ctx, err)
		return
	}
	if !isFile {
		pp.ErrorLog(ctx, "the provided path indicates a directory, not a file")
		return
	}
	encryptionTag := ""
	if isEncrypted {
		encryptionTag = utils.GetRandomString(8)
	}
	uploadFileHandler := GetUploadFileHandler(isVideoStream)
	fileInfo, slices, err := uploadFileHandler.PreUpload(ctx, path, encryptionTag)
	if err != nil {
		pp.ErrorLog(ctx, "failed to slice file before upload ", err)
		return
	}

	reqTime := time.Now().Unix()
	p := requests.RequestUploadFileData(ctx, fileInfo, slices, desiredTier, allowHigherTier, walletAddr, walletPubkey, wsign, reqTime)
	if err = ReqGetWalletOzForUpload(ctx, setting.WalletAddress, task.LOCAL_REQID, p); err != nil {
		pp.ErrorLog(ctx, err)
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
		Address:  p2pserver.GetP2pServer(ctx).GetPPInfo(),
	}
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, p, header.ReqFileBackupStatus)
}

// RspUploadFile response of upload file event, SP -> upgrader, upgrader -> dest PP
func RspUploadFile(ctx context.Context, _ core.WriteCloser) {
	pp.DebugLog(ctx, "get RspUploadFile")
	target := &protos.RspUploadFile{}
	if err := VerifyMessage(ctx, header.RspUploadFile, target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, target) {
		pp.ErrorLog(ctx, "unmarshal error")
		return
	}

	metrics.UploadPerformanceLogNow(target.FileHash + ":RCV_RSP_UPLOAD_SP")

	// SPAM check
	if time.Now().Unix()-target.TimeStamp > setting.SpamThresholdSpSignLatency {
		pp.ErrorLog(ctx, "sp's upload file response was expired")
		return
	}

	// upload file to PP based on the PP info provided by SP
	if target.Result == nil {
		pp.ErrorLog(ctx, "target.Result is nil")
		return
	}

	if target.Result.State != protos.ResultState_RES_SUCCESS {
		if strings.Contains(target.Result.Msg, "Same file with the name") {
			pp.ErrorLog(ctx, target.Result.Msg)
		} else {
			pp.ErrorLog(ctx, "upload failed: ", target.Result.Msg)
		}

		if file.IsFileRpcRemote(target.FileHash) {
			file.SetRemoteFileResult(target.FileHash, rpc.Result{Return: target.Result.Msg})
		} else {
			file.ClearFileMap(target.FileHash)
		}
		return
	}

	task.UploadTaskIdMap.Store(target.FileHash, target.TaskId)

	if len(target.Slices) != 0 {
		go startUploadTask(ctx, target)
	} else {
		pp.Log(ctx, "file upload successful！  fileHash", target.FileHash)
		//var p float32 = 100
		//ProgressMap.Store(target.FileHash, p)
		task.UploadProgressMap.Delete(target.FileHash)
	}

	// tell the rpc client, uploading to sds network has successfully started.
	if file.IsFileRpcRemote(target.FileHash) {
		file.SetRemoteFileResult(target.FileHash, rpc.Result{Return: rpc.SUCCESS})
	}

	if isCover {
		pp.DebugLog(ctx, "is_cover")
	}
}

func RspBackupStatus(ctx context.Context, _ core.WriteCloser) {
	pp.DebugLog(ctx, "get RspBackupStatus")
	target := &protos.RspBackupStatus{}
	if err := VerifyMessage(ctx, header.RspFileBackupStatus, target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, target) {
		pp.ErrorLog(ctx, "unmarshal error")
		return
	}

	// SPAM check
	if time.Now().Unix()-target.TimeStamp > setting.SpamThresholdSpSignLatency {
		pp.ErrorLog(ctx, "sp's backup file response was expired,", time.Now().Unix(), ",", target.TimeStamp)
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

	if len(target.Slices) == 0 {
		ScheduleReqBackupStatus(ctx, target.FileHash)
		return
	}

	pp.Logf(ctx, "Start re-uploading slices for the file  %s", target.FileHash)
	totalSize := int64(0)
	for _, slice := range target.Slices {
		totalSize += int64(slice.GetSliceSize())
	}
	uploadTask := task.CreateBackupFileTask(target)
	p2pserver.GetP2pServer(ctx).CleanUpConnMap("upload#" + target.FileHash)
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
	// Create upload task
	uploadTask := task.CreateUploadFileTask(target)

	p2pserver.GetP2pServer(ctx).CleanUpConnMap(target.FileHash)
	task.UploadFileTaskMap.Store(target.FileHash, uploadTask)

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
			utils.DebugLog("Fatal error")
			return err
		}

		// Report slice failures to SP to get assigned new slice destinations
		slicesToReDownload, failedSlices := uploadTask.SliceFailuresToReport()
		if len(slicesToReDownload) > 0 {
			if !uploadTask.CanRetry() {
				return errors.New("max upload retry count reached")
			}
			p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, requests.ReqUploadSlicesWrong(ctx, uploadTask, uploadTask.RspUploadFile.SpP2PAddress, slicesToReDownload, failedSlices), header.ReqUploadSlicesWrong)
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
			uploadSliceTask, err = task.CreateUploadSliceTask(ctx, slice, uploadTask)
			if err != nil {
				slice.SetError(err, true, uploadTask)
				return
			}
			pp.DebugLogf(ctx, "starting to upload slice %v for file %v", slice.Slice.SliceNumber, uploadTask.RspUploadFile.FileHash)
			err = UploadFileSlice(ctx, uploadSliceTask)
			if err != nil {
				utils.ErrorLogf("Error uploading slice %v: %v", uploadSliceTask.SliceHash, err.Error())
				slice.SetError(err, false, uploadTask)
				return
			}
		case protos.UploadType_BACKUP:
			uploadSliceTask, err = task.GetReuploadSliceTask(ctx, slice, destination.PpInfo, uploadTask)
			if err != nil {
				slice.SetError(err, true, uploadTask)
				return
			}
			pp.DebugLogf(ctx, "starting to backup slice %v for file %v", slice.Slice.SliceNumber, uploadTask.RspUploadFile.FileHash)
			err = BackupFileSlice(ctx, uploadSliceTask)
			if err != nil {
				slice.SetError(err, false, uploadTask)
				return
			}
		}

		slice.SetStatus(task.SLICE_STATUS_FINISHED, uploadTask)
	}
}

func UploadPause(ctx context.Context, fileHash, reqID string, w http.ResponseWriter) {
	p2pserver.GetP2pServer(ctx).RangeCachedConn("upload#"+fileHash, func(k, v interface{}) bool {
		conn := v.(*cf.ClientConn)
		conn.ClientClose(true)
		pp.DebugLog(ctx, "UploadPause", conn)
		return true
	})
	p2pserver.GetP2pServer(ctx).CleanUpConnMap(fileHash)
	task.UploadFileTaskMap.Delete(fileHash)
	task.UploadProgressMap.Delete(fileHash)
}

func encryptSliceData(rawData []byte) ([]byte, error) {
	hdKeyNonce := rand.Uint32()
	if hdKeyNonce > hdkey.HardenedKeyStart {
		hdKeyNonce -= hdkey.HardenedKeyStart
	}
	aesNonce := rand.Uint64()

	key, err := hdkey.MasterKeyForSliceEncryption(setting.WalletPrivateKey, hdKeyNonce)
	if err != nil {
		return nil, err
	}

	encryptedData, err := encryption.EncryptAES(key.PrivateKey(), rawData, aesNonce)
	if err != nil {
		return nil, err
	}

	encryptedSlice := &protos.EncryptedSlice{
		HdkeyNonce: hdKeyNonce,
		AesNonce:   aesNonce,
		Data:       encryptedData,
		RawSize:    uint64(len(rawData)),
	}
	return proto.Marshal(encryptedSlice)
}

func GetUploadFileHandler(isVideoStream bool) UploadFileHandler {
	if isVideoStream {
		return UploadStreamFileHandler{}
	}
	return UploadRawFileHandler{}
}

type UploadFileHandler interface {
	PreUpload(ctx context.Context, filePath, encryptionTag string) (*protos.FileInfo, []*protos.SliceHashAddr, error)
}

type UploadStreamFileHandler struct {
}

type UploadRawFileHandler struct {
}

func (UploadStreamFileHandler) PreUpload(ctx context.Context, filePath, encryptionTag string) (*protos.FileInfo, []*protos.SliceHashAddr, error) {
	info, err := file.GetFileInfo(filePath)
	if err != nil {
		pp.ErrorLog(ctx, "wrong filePath", err.Error())
		return nil, nil, err
	}

	duration, err := file.GetVideoDuration(filePath)
	if err != nil {
		pp.ErrorLog(ctx, "Failed to get the length of the video: ", err)
		return nil, nil, err
	}

	fileName := info.Name()
	fileSize := uint64(info.Size())
	fileHash := file.GetFileHashForVideoStream(filePath, encryptionTag)

	sliceSize := setting.DefaultSliceBlockSize

	var sliceDuration float64
	sliceDuration = math.Floor(float64(duration) * float64(sliceSize) / float64(fileSize))
	sliceDuration = math.Min(float64(setting.DefaultHlsSegmentLength), sliceDuration)
	sliceCount := uint64(math.Ceil(float64(duration)/sliceDuration)) + setting.DefaultHlsSegmentBuffer + 1

	file.VideoToHls(ctx, fileHash, file.GetFilePath(fileHash))

	hlsInfo, err := file.GetHlsInfo(fileHash, sliceCount)
	if err != nil {
		pp.ErrorLog(ctx, "Hls transformation failed: ", err)
		return nil, nil, err
	}
	videoFolder := file.GetVideoTmpFolder(fileHash)

	var totalSize int64
	var slices []*protos.SliceHashAddr
	for sliceNumber := uint64(1); sliceNumber <= sliceCount; sliceNumber++ {
		var rawData []byte
		var sliceSize int64
		if sliceNumber == 1 {
			jsonStr, _ := json.Marshal(hlsInfo)
			rawData = jsonStr
			sliceSize = int64(len(rawData))
		} else if sliceNumber < hlsInfo.StartSliceNumber {
			rawData = file.GetDumpySliceData(fileHash, sliceNumber)
			sliceSize = int64(len(rawData))
		} else {
			sliceName := hlsInfo.SliceToSegment[sliceNumber]
			slicePath := videoFolder + "/" + sliceName
			fileInfo, err := file.GetFileInfo(slicePath)
			if err != nil {
				return nil, nil, errors.New("wrong file path")
			}
			rawData, err = file.GetWholeFileData(slicePath)
			if err != nil {
				return nil, nil, errors.New("failed getting whole file data")
			}
			sliceSize = fileInfo.Size()
		}

		data := rawData
		if encryptionTag != "" {
			data, err = encryptSliceData(rawData)
			if err != nil {
				return nil, nil, errors.Wrap(err, "Couldn't encrypt slice data")
			}
		}
		sliceHash := utils.CalcSliceHash(data, fileHash, sliceNumber)
		err = file.SaveTmpSliceData(fileHash, sliceHash, data)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to save to temp file")
		}

		slice := &protos.SliceHashAddr{
			SliceHash:   sliceHash,
			SliceNumber: sliceNumber,
			SliceSize:   uint64(sliceSize),
			SliceOffset: &protos.SliceOffset{
				SliceOffsetStart: 0,
				SliceOffsetEnd:   uint64(sliceSize),
			},
		}
		slices = append(slices, slice)
		totalSize += sliceSize
		err := file.SaveTmpSliceData(fileHash, sliceHash, data)
		if err != nil {
			return nil, nil, err
		}
	}
	file.DeleteTmpHlsFolder(ctx, fileHash)
	fileInfo := &protos.FileInfo{
		FileSize:           uint64(totalSize),
		FileName:           fileName,
		FileHash:           fileHash,
		StoragePath:        filePath,
		EncryptionTag:      encryptionTag,
		Duration:           duration,
		OwnerWalletAddress: setting.WalletAddress,
	}

	return fileInfo, slices, nil
}

func (UploadRawFileHandler) PreUpload(ctx context.Context, filePath, encryptionTag string) (*protos.FileInfo, []*protos.SliceHashAddr, error) {
	info, err := file.GetFileInfo(filePath)
	if err != nil {
		pp.ErrorLog(ctx, "wrong filePath", err.Error())
		return nil, nil, err
	}
	fileName := info.Name()
	fileSize := uint64(info.Size())
	fileHash := file.GetFileHash(filePath, encryptionTag)
	sliceSize := uint64(setting.DefaultSliceBlockSize)
	sliceCount := uint64(math.Ceil(float64(info.Size()) / float64(sliceSize)))

	var slices []*protos.SliceHashAddr
	for sliceNumber := uint64(1); sliceNumber <= sliceCount; sliceNumber++ {
		sliceOffset := requests.GetSliceOffset(sliceNumber, sliceCount, sliceSize, fileSize)

		rawData, err := file.GetFileData(filePath, sliceOffset)
		if err != nil {
			return nil, nil, errors.New("Failed reading data from file")

		}

		// Encrypt slice data if required
		data := rawData
		if encryptionTag != "" {
			data, err = encryptSliceData(rawData)
			if err != nil {
				return nil, nil, errors.Wrap(err, "Couldn't encrypt slice data")
			}
		}
		sliceHash := utils.CalcSliceHash(data, fileHash, sliceNumber)
		err = file.SaveTmpSliceData(fileHash, sliceHash, data)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to save to temp file")
		}

		SliceHashAddr := &protos.SliceHashAddr{
			SliceHash:   sliceHash,
			SliceSize:   sliceOffset.SliceOffsetEnd - sliceOffset.SliceOffsetStart,
			SliceNumber: sliceNumber,
			SliceOffset: sliceOffset,
		}

		slices = append(slices, SliceHashAddr)
	}

	fileInfo := &protos.FileInfo{
		FileSize:           uint64(info.Size()),
		FileName:           fileName,
		FileHash:           fileHash,
		StoragePath:        filePath,
		EncryptionTag:      encryptionTag,
		OwnerWalletAddress: setting.WalletAddress,
	}

	return fileInfo, slices, nil
}
