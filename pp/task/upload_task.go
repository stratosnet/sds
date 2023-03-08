package task

import (
	"context"
	"encoding/json"
	"math/rand"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/metrics"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/encryption"
	"github.com/stratosnet/sds/utils/encryption/hdkey"
	"google.golang.org/protobuf/proto"
)

// UploadSliceTask represents a slice upload task that is in progress
type UploadSliceTask struct {
	TaskID          string
	FileHash        string
	SliceNumAddr    *protos.SliceNumAddr // upload PP address and sliceNumber
	SliceOffsetInfo *protos.SliceOffsetInfo
	FileCRC         uint32
	Data            []byte
	SliceTotalSize  uint64
	SpP2pAddress    string
}

const (
	SLICE_STATUS_NOT_STARTED = iota
	SLICE_STATUS_FAILED
	SLICE_STATUS_WAITING_FOR_SP
	SLICE_STATUS_REPLACED
	SLICE_STATUS_FINISHED

	MAXSLICE              = 50 // max number of slices that can upload concurrently for a single file
	UPLOAD_TIMER_INTERVAL = 10 // seconds
	MAX_UPLOAD_RETRY      = 5
)

// UploadFileTask represents a file upload task that is in progress
type UploadFileTask struct {
	FileCRC       uint32
	FileHash      string
	IsEncrypted   bool
	IsVideoStream bool
	Sign          []byte
	Slices        map[string]*SlicesPerDestination
	SpP2pAddress  string
	TaskID        string
	Type          protos.UploadType

	ConcurrentUploads int
	FatalError        error
	RetryCount        int
	UpChan            chan bool
	Mutex             sync.RWMutex
}

type SlicesPerDestination struct {
	PpInfo  *protos.PPBaseInfo
	Slices  []*SliceWithStatus
	Started bool
}

type SliceWithStatus struct {
	Error       error
	Fatal       bool // Whether this error should cancel the whole file upload or not
	SliceHash   string
	SliceNumber uint64
	SliceOffset *protos.SliceOffset
	SliceSize   uint64
	Status      int
	SpNodeSign  []byte
	CostTime    int64
}

func CreateUploadFileTask(fileHash, taskId, spP2pAddress string, isEncrypted, isVideoStream bool, signature []byte, slices []*protos.SliceHashAddr, uploadType protos.UploadType) *UploadFileTask {
	task := &UploadFileTask{
		FileCRC:           utils.CalcFileCRC32(file.GetFilePath(fileHash)),
		FileHash:          fileHash,
		IsEncrypted:       isEncrypted,
		IsVideoStream:     isVideoStream,
		Sign:              signature,
		Slices:            make(map[string]*SlicesPerDestination),
		SpP2pAddress:      spP2pAddress,
		TaskID:            taskId,
		Type:              uploadType,
		ConcurrentUploads: 0,
		RetryCount:        0,
		UpChan:            make(chan bool, MAXSLICE),
		Mutex:             sync.RWMutex{},
	}

	for _, slice := range slices {
		task.addNewSlice(slice)
	}
	metrics.TaskCount.WithLabelValues("upload").Inc()
	return task
}

func (u *UploadFileTask) addNewSlice(slice *protos.SliceHashAddr) {
	slicesPerDestination := u.Slices[slice.PpInfo.P2PAddress]
	if slicesPerDestination == nil {
		slicesPerDestination = &SlicesPerDestination{
			PpInfo:  slice.PpInfo,
			Started: false,
		}
		u.Slices[slice.PpInfo.P2PAddress] = slicesPerDestination
	}

	slicesPerDestination.Slices = append(slicesPerDestination.Slices, &SliceWithStatus{
		SliceHash:   slice.SliceHash,
		SliceNumber: slice.SliceNumber,
		SliceOffset: slice.SliceOffset,
		SliceSize:   slice.SliceSize,
		Status:      SLICE_STATUS_NOT_STARTED,
		SpNodeSign:  slice.SpNodeSign,
	})
}

func (u *UploadFileTask) SignalNewDestinations() {
	u.Mutex.RLock()
	defer u.Mutex.RUnlock()

	for _, destination := range u.Slices {
		if !destination.Started {
			select {
			case u.UpChan <- true:
			default: // channel is already full
			}
		}
	}
}

func (u *UploadFileTask) IsFinished() bool {
	u.Mutex.RLock()
	defer u.Mutex.RUnlock()

	for _, destination := range u.Slices {
		for _, slice := range destination.Slices {
			if slice.Status != SLICE_STATUS_FINISHED && slice.Status != SLICE_STATUS_REPLACED {
				return false
			}
		}
	}

	return true
}

func (u *UploadFileTask) IsFatal() error {
	u.Mutex.RLock()
	defer u.Mutex.RUnlock()

	if u.FatalError != nil {
		return u.FatalError
	}
	for _, slicesPerDestination := range u.Slices {
		for _, slice := range slicesPerDestination.Slices {
			if slice.Fatal {
				return slice.Error
			}
		}
	}

	return nil
}

// SliceFailuresToReport returns the list of slices that will require a new destination, and a boolean list of the same length indicating which slices actually failed
func (u *UploadFileTask) SliceFailuresToReport() ([]*protos.SliceHashAddr, []bool) {
	u.Mutex.Lock()
	defer u.Mutex.Unlock()

	var slicesToReDownload []*protos.SliceHashAddr
	var failedSlices []bool
	for _, slicesPerDestination := range u.Slices {
		errorPresent := false
		for _, slice := range slicesPerDestination.Slices {
			if slice.Status == SLICE_STATUS_FAILED {
				errorPresent = true
			}
		}

		if !errorPresent {
			continue
		}

		// There was an error sending slices to this destination, so all associated failed and not started slices will receive a new destination PP
		for _, slice := range slicesPerDestination.Slices {
			if slice.Status == SLICE_STATUS_FAILED || slice.Status == SLICE_STATUS_NOT_STARTED || slice.Status == SLICE_STATUS_WAITING_FOR_SP {
				slicesToReDownload = append(slicesToReDownload, &protos.SliceHashAddr{
					SliceHash:   slice.SliceHash,
					SliceNumber: slice.SliceNumber,
					SliceOffset: slice.SliceOffset,
					SliceSize:   slice.SliceSize,
					PpInfo:      slicesPerDestination.PpInfo,
				})
				failedSlices = append(failedSlices, slice.Status == SLICE_STATUS_FAILED)
				slice.Status = SLICE_STATUS_WAITING_FOR_SP
			}
		}
	}

	return slicesToReDownload, failedSlices
}

func (u *UploadFileTask) CanRetry() bool {
	return u.RetryCount < MAX_UPLOAD_RETRY
}

func (u *UploadFileTask) GetExcludedDestinations() []*protos.PPBaseInfo {
	u.Mutex.RLock()
	defer u.Mutex.RUnlock()

	var destinations []*protos.PPBaseInfo
	for _, destination := range u.Slices {
		for _, slice := range destination.Slices {
			if slice.Status == SLICE_STATUS_FAILED || slice.Status == SLICE_STATUS_WAITING_FOR_SP || slice.Status == SLICE_STATUS_REPLACED {
				destinations = append(destinations, destination.PpInfo)
				break
			}
		}
	}

	return destinations
}

func (u *UploadFileTask) NextDestination() *SlicesPerDestination {
	u.Mutex.Lock()
	defer u.Mutex.Unlock()

	if u.ConcurrentUploads >= MAXSLICE {
		return nil
	}
	for _, destination := range u.Slices {
		if !destination.Started {
			destination.Started = true
			u.ConcurrentUploads++
			return destination
		}
	}

	return nil
}

func (u *UploadFileTask) UpdateSliceDestinations(newDestinations []*protos.SliceHashAddr) {
	u.Mutex.Lock()
	defer u.Mutex.Unlock()

	// Get original destination for each slice
	originalDestinations := make(map[uint64]string)
	for p2pAddress, destination := range u.Slices {
		for _, slice := range destination.Slices {
			originalDestinations[slice.SliceNumber] = p2pAddress
		}
	}

	// Update slice destinations
	for _, newDestination := range newDestinations {
		originalP2pAddress, ok := originalDestinations[newDestination.SliceNumber]
		if !ok {
			continue
		}

		slicesOriginalDestination := u.Slices[originalP2pAddress]
		if slicesOriginalDestination == nil {
			continue
		}
		for _, slice := range slicesOriginalDestination.Slices {
			if slice.SliceNumber == newDestination.SliceNumber {
				slice.Status = SLICE_STATUS_REPLACED
				u.addNewSlice(newDestination)
				break
			}
		}
	}
}

func (s *SliceWithStatus) SetError(err error, fatal bool, uploadTask *UploadFileTask) {
	uploadTask.Mutex.Lock()
	defer uploadTask.Mutex.Unlock()

	s.Error = err
	s.Fatal = fatal
	s.Status = SLICE_STATUS_FAILED
}

func (s *SliceWithStatus) SetStatus(status int, uploadTask *UploadFileTask) {
	uploadTask.Mutex.Lock()
	defer uploadTask.Mutex.Unlock()

	s.Status = status
}

// UploadFileTaskMap Map of file upload tasks that are in progress.
var UploadFileTaskMap = &sync.Map{} // map[string]*UploadFileTask

// UploadProgress represents the progress for an ongoing upload
type UploadProgress struct {
	Total     int64
	HasUpload int64
}

// UploadProgressMap Map of the progress for ongoing uploads
var UploadProgressMap = &sync.Map{} // map[string]*UploadProgress

func CreateUploadSliceTask(ctx context.Context, slice *SliceWithStatus, ppInfo *protos.PPBaseInfo, uploadTask *UploadFileTask) (*UploadSliceTask, error) {
	if uploadTask.IsVideoStream {
		return CreateUploadSliceTaskStream(ctx, slice, ppInfo, uploadTask)
	} else {
		return CreateUploadSliceTaskFile(ctx, slice, ppInfo, uploadTask)
	}
}

func CreateUploadSliceTaskFile(ctx context.Context, slice *SliceWithStatus, ppInfo *protos.PPBaseInfo, uploadTask *UploadFileTask) (*UploadSliceTask, error) {
	pp.DebugLogf(ctx, "sliceNumber %v  offsetStart = %v  offsetEnd = %v", slice.SliceNumber, slice.SliceOffset.SliceOffsetStart, slice.SliceOffset.SliceOffsetEnd)
	startOffset := slice.SliceOffset.SliceOffsetStart
	endOffset := slice.SliceOffset.SliceOffsetEnd

	var fileSize uint64
	var filePath string

	remote := file.IsFileRpcRemote(uploadTask.FileHash)
	if !remote {
		// in case of local file
		filePath = file.GetFilePath(uploadTask.FileHash)
		fileInfo, err := file.GetFileInfo(filePath)
		if fileInfo == nil {
			return nil, errors.Wrap(err, "wrong file path")
		}
		fileSize = uint64(fileInfo.Size())
	} else {
		// in case of remote (rpc) file
		fileSize = file.GetRemoteFileSize(uploadTask.FileHash)
	}

	if fileSize < endOffset {
		endOffset = fileSize
	}
	offset := &protos.SliceOffset{
		SliceOffsetStart: startOffset,
		SliceOffsetEnd:   endOffset,
	}

	var rawData []byte
	var err error
	tmpFileName := uuid.NewString()
	if !remote {
		metrics.UploadPerformanceLogNow(uploadTask.FileHash + ":SND_GET_LOCAL_DATA:" + strconv.FormatInt(int64(offset.SliceOffsetStart), 10))
		rawData, err = file.GetFileData(filePath, offset)
		if err != nil {
			return nil, errors.Wrap(err, "failed getting file data")
		}
		if rawData != nil {
			err = file.SaveTmpSliceData(uploadTask.FileHash, tmpFileName, rawData)
			if err != nil {
				return nil, errors.Wrap(err, "filed saving tmp slice data")
			}
		}
		metrics.UploadPerformanceLogNow(uploadTask.FileHash + ":RCV_GET_LOCAL_DATA:" + strconv.FormatInt(int64(offset.SliceOffsetStart), 10))
	} else {
		metrics.UploadPerformanceLogNow(uploadTask.FileHash + ":SND_GET_REMOTE_DATA:" + strconv.FormatInt(int64(offset.SliceOffsetStart), 10))
		if file.CacheRemoteFileData(uploadTask.FileHash, offset, tmpFileName) == nil {
			rawData, err = file.GetSliceDataFromTmp(uploadTask.FileHash, tmpFileName)
			if err != nil {
				return nil, errors.Wrap(err, "failed getting slice data from tmp")
			}
		}
		metrics.UploadPerformanceLogNow(uploadTask.FileHash + ":RCV_GET_REMOTE_DATA:" + strconv.FormatInt(int64(offset.SliceOffsetStart), 10))
	}

	if rawData == nil {
		return nil, errors.New("Failed reading data from file")
	}

	// Encrypt slice data if required
	data := rawData
	if uploadTask.IsEncrypted {
		var err error
		data, err = encryptSliceData(rawData)
		if err != nil {
			return nil, errors.Wrap(err, "Couldn't encrypt slice data")
		}
		// write data back to the tmp file
		err = file.SaveTmpSliceData(uploadTask.FileHash, tmpFileName, data)
		if err != nil {
			return nil, err
		}
	}
	dataSize := uint64(len(data))
	sliceHash := utils.CalcSliceHash(data, uploadTask.FileHash, slice.SliceNumber)

	sl := &protos.SliceOffsetInfo{
		SliceHash: sliceHash,
		SliceOffset: &protos.SliceOffset{
			SliceOffsetStart: 0,
			SliceOffsetEnd:   dataSize,
		},
	}

	sliceNumAddr := &protos.SliceNumAddr{
		SliceNumber: slice.SliceNumber,
		SliceOffset: slice.SliceOffset,
		PpInfo:      ppInfo,
		SpNodeSign:  slice.SpNodeSign,
	}
	tk := &UploadSliceTask{
		TaskID:          uploadTask.TaskID,
		FileHash:        uploadTask.FileHash,
		SliceNumAddr:    sliceNumAddr,
		SliceOffsetInfo: sl,
		FileCRC:         uploadTask.FileCRC,
		SliceTotalSize:  dataSize,
		SpP2pAddress:    uploadTask.SpP2pAddress,
	}

	err = file.RenameTmpFile(uploadTask.FileHash, tmpFileName, sliceHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed renaming tmp file")
	}
	return tk, nil
}

func CreateUploadSliceTaskStream(ctx context.Context, slice *SliceWithStatus, ppInfo *protos.PPBaseInfo, uploadTask *UploadFileTask) (*UploadSliceTask, error) {
	videoFolder := file.GetVideoTmpFolder(uploadTask.FileHash)
	videoSliceInfo := file.HlsInfoMap[uploadTask.FileHash]
	var data []byte
	var sliceTotalSize uint64
	if slice.SliceNumber == 1 {
		jsonStr, _ := json.Marshal(videoSliceInfo)
		data = jsonStr
		sliceTotalSize = uint64(len(data))
	} else if slice.SliceNumber < videoSliceInfo.StartSliceNumber {
		data = file.GetDumpySliceData(uploadTask.FileHash, slice.SliceNumber)
		sliceTotalSize = uint64(len(data))
	} else {
		sliceName := videoSliceInfo.SliceToSegment[slice.SliceNumber]
		slicePath := videoFolder + "/" + sliceName
		fileInfo, err := file.GetFileInfo(slicePath)
		if err != nil {
			return nil, errors.New("wrong file path")
		}
		data, err = file.GetWholeFileData(slicePath)
		if err != nil {
			return nil, errors.New("failed getting whole file data")
		}
		sliceTotalSize = uint64(fileInfo.Size())
	}

	pp.DebugLog(ctx, "sliceNumber", slice.SliceNumber)

	sliceHash := utils.CalcSliceHash(data, uploadTask.FileHash, slice.SliceNumber)
	offset := &protos.SliceOffset{
		SliceOffsetStart: uint64(0),
		SliceOffsetEnd:   sliceTotalSize,
	}
	sl := &protos.SliceOffsetInfo{
		SliceHash:   sliceHash,
		SliceOffset: offset,
	}
	SliceNumAddr := &protos.SliceNumAddr{
		SliceNumber: slice.SliceNumber,
		SliceOffset: offset,
		PpInfo:      ppInfo,
		SpNodeSign:  slice.SpNodeSign,
	}
	slice.SliceOffset = offset
	tk := &UploadSliceTask{
		TaskID:          uploadTask.TaskID,
		FileHash:        uploadTask.FileHash,
		SliceNumAddr:    SliceNumAddr,
		SliceOffsetInfo: sl,
		FileCRC:         uploadTask.FileCRC,
		Data:            data,
		SliceTotalSize:  sliceTotalSize,
		SpP2pAddress:    uploadTask.SpP2pAddress,
	}

	err := file.SaveTmpSliceData(uploadTask.FileHash, sliceHash, data)
	if err != nil {
		return nil, err
	}
	return tk, nil
}

func GetReuploadSliceTask(ctx context.Context, slice *SliceWithStatus, ppInfo *protos.PPBaseInfo, uploadTask *UploadFileTask) (*UploadSliceTask, error) {
	pp.DebugLogf(ctx, "  fileHash %s sliceNumber %v, sliceHash %s",
		uploadTask.FileHash, slice.SliceNumber, slice.SliceHash)

	rawData, err := file.GetSliceDataFromTmp(uploadTask.FileHash, slice.SliceHash)

	if rawData == nil {
		return nil, errors.Wrapf(err, "Failed to find the file slice in temp folder for fileHash %s sliceNumber %v, sliceHash %s",
			uploadTask.FileHash, slice.SliceNumber, slice.SliceHash)
	}

	data := rawData
	dataSize := uint64(len(data))

	sl := &protos.SliceOffsetInfo{
		SliceHash: slice.SliceHash,
		SliceOffset: &protos.SliceOffset{
			SliceOffsetStart: 0,
			SliceOffsetEnd:   dataSize,
		},
	}

	tk := &UploadSliceTask{
		TaskID:   uploadTask.TaskID,
		FileHash: uploadTask.FileHash,
		SliceNumAddr: &protos.SliceNumAddr{
			SliceNumber: slice.SliceNumber,
			SliceOffset: slice.SliceOffset,
			PpInfo:      ppInfo,
		},
		SliceOffsetInfo: sl,
		Data:            data,
		SliceTotalSize:  dataSize,
		SpP2pAddress:    uploadTask.SpP2pAddress,
	}
	return tk, nil
}

func SaveUploadFile(target *protos.ReqUploadFileSlice) error {
	return file.SaveSliceData(target.Data, target.SliceInfo.SliceHash, target.SliceInfo.SliceOffset.SliceOffsetStart)
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
