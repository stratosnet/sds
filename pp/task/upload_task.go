package task

import (
	"context"
	"math/rand"
	"strconv"
	"sync"

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
	RspUploadFile *protos.RspUploadFile
	RspBackupFile *protos.RspBackupStatus
	SliceNumber   uint64
	SliceHash     string
	Type          protos.UploadType
	Data          []byte
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
	RspUploadFile     *protos.RspUploadFile
	RspBackupFile     *protos.RspBackupStatus
	Type              protos.UploadType
	FileCRC           uint32
	Destinations      map[string]*SlicesPerDestination
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

// SliceWithStatus wraps a SliceHashAddr, and it provides extra states for upload/backup task.
type SliceWithStatus struct {
	Slice    *protos.SliceHashAddr
	Error    error
	Fatal    bool // Whether this error should cancel the whole file upload or not
	Status   int
	CostTime int64
}

func CreateBackupFileTask(target *protos.RspBackupStatus) *UploadFileTask {
	if target == nil {
		return nil
	}

	task := &UploadFileTask{
		RspUploadFile:     nil,
		RspBackupFile:     target,
		FileCRC:           utils.CalcFileCRC32(file.GetFilePath(target.FileHash)),
		Type:              protos.UploadType_BACKUP,
		Destinations:      make(map[string]*SlicesPerDestination),
		ConcurrentUploads: 0,
		RetryCount:        0,
		UpChan:            make(chan bool, MAXSLICE),
		Mutex:             sync.RWMutex{},
	}

	for _, slice := range target.Slices {
		_, ok := task.Destinations[slice.PpInfo.P2PAddress]
		if !ok {
			task.Destinations[slice.PpInfo.P2PAddress] = &SlicesPerDestination{
				PpInfo:  slice.PpInfo,
				Started: false,
			}
		}
		sws := &SliceWithStatus{
			Slice:  slice,
			Status: SLICE_STATUS_NOT_STARTED,
		}
		task.Destinations[slice.PpInfo.P2PAddress].Slices = append(task.Destinations[slice.PpInfo.P2PAddress].Slices, sws)
	}
	metrics.TaskCount.WithLabelValues("upload").Inc()
	return task
}

func CreateUploadFileTask(target *protos.RspUploadFile) *UploadFileTask {
	if target == nil {
		return nil
	}

	task := &UploadFileTask{
		RspUploadFile:     target,
		RspBackupFile:     nil,
		FileCRC:           utils.CalcFileCRC32(file.GetFilePath(target.FileHash)),
		Type:              protos.UploadType_NEW_UPLOAD,
		Destinations:      make(map[string]*SlicesPerDestination),
		ConcurrentUploads: 0,
		RetryCount:        0,
		UpChan:            make(chan bool, MAXSLICE),
		Mutex:             sync.RWMutex{},
	}

	for _, slice := range target.Slices {
		_, ok := task.Destinations[slice.PpInfo.P2PAddress]
		if !ok {
			task.Destinations[slice.PpInfo.P2PAddress] = &SlicesPerDestination{
				PpInfo:  slice.PpInfo,
				Started: false,
			}
		}
		sws := &SliceWithStatus{
			Slice:  slice,
			Status: SLICE_STATUS_NOT_STARTED,
		}
		task.Destinations[slice.PpInfo.P2PAddress].Slices = append(task.Destinations[slice.PpInfo.P2PAddress].Slices, sws)
	}
	metrics.TaskCount.WithLabelValues("upload").Inc()
	return task
}

func (u *UploadFileTask) addNewSlice(slice *protos.SliceHashAddr) {
	slicesPerDestination := u.Destinations[slice.PpInfo.P2PAddress]
	if slicesPerDestination == nil {
		slicesPerDestination = &SlicesPerDestination{
			PpInfo:  slice.PpInfo,
			Started: false,
		}
		u.Destinations[slice.PpInfo.P2PAddress] = slicesPerDestination
	}

	slicesPerDestination.Slices = append(slicesPerDestination.Slices, &SliceWithStatus{
		Slice:  slice,
		Status: SLICE_STATUS_NOT_STARTED,
	})
}

func (u *UploadFileTask) SignalNewDestinations() {
	u.Mutex.RLock()
	defer u.Mutex.RUnlock()

	for _, destination := range u.Destinations {
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

	for _, destination := range u.Destinations {
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
	for _, slicesPerDestination := range u.Destinations {
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
	for _, slicesPerDestination := range u.Destinations {
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
				slicesToReDownload = append(slicesToReDownload, slice.Slice)
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
	for _, destination := range u.Destinations {
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
	for _, destination := range u.Destinations {
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
	for p2pAddress, destination := range u.Destinations {
		for _, slice := range destination.Slices {
			originalDestinations[slice.Slice.SliceNumber] = p2pAddress
		}
	}

	// Update slice destinations
	for _, newDestination := range newDestinations {
		originalP2pAddress, ok := originalDestinations[newDestination.SliceNumber]
		if !ok {
			continue
		}

		slicesOriginalDestination := u.Destinations[originalP2pAddress]
		if slicesOriginalDestination == nil {
			continue
		}
		for _, slice := range slicesOriginalDestination.Slices {
			if slice.Slice.SliceNumber == newDestination.SliceNumber {
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

func CreateUploadSliceTask(ctx context.Context, slice *SliceWithStatus, uploadTask *UploadFileTask,
	sliceDataAccessor func(fileHash string, slice *protos.SliceHashAddr) ([]byte, error)) (*UploadSliceTask, error) {
	pp.DebugLogf(ctx, "sliceNumber %v  offsetStart = %v  offsetEnd = %v", slice.Slice.SliceNumber, slice.Slice.SliceOffset.SliceOffsetStart, slice.Slice.SliceOffset.SliceOffsetEnd)
	startOffset := slice.Slice.SliceOffset.SliceOffsetStart
	endOffset := slice.Slice.SliceOffset.SliceOffsetEnd
	fileHash := uploadTask.RspUploadFile.FileHash

	var fileSize uint64
	var filePath string

	remote := file.IsFileRpcRemote(fileHash)
	if !remote {
		// in case of local file
		filePath = file.GetFilePath(fileHash)
		fileInfo, err := file.GetFileInfo(filePath)
		if fileInfo == nil {
			return nil, errors.Wrap(err, "wrong file path")
		}
		fileSize = uint64(fileInfo.Size())
	} else {
		// in case of remote (rpc) file
		fileSize = file.GetRemoteFileSize(fileHash)
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
	tmpFileName := strconv.FormatUint(slice.Slice.SliceNumber, 10)
	if !remote {
		metrics.UploadPerformanceLogNow(fileHash + ":SND_GET_LOCAL_DATA:" + strconv.FormatInt(int64(offset.SliceOffsetStart), 10))
		rawData, err = sliceDataAccessor(fileHash, slice.Slice)
		if err != nil {
			return nil, errors.Wrap(err, "failed getting file data")
		}
		if rawData != nil {
			err = file.SaveTmpSliceData(fileHash, tmpFileName, rawData)
			if err != nil {
				return nil, errors.Wrap(err, "filed saving tmp slice data")
			}
		}
		metrics.UploadPerformanceLogNow(fileHash + ":RCV_GET_LOCAL_DATA:" + strconv.FormatInt(int64(offset.SliceOffsetStart), 10))
	} else {
		metrics.UploadPerformanceLogNow(fileHash + ":SND_GET_REMOTE_DATA:" + strconv.FormatInt(int64(offset.SliceOffsetStart), 10))
		if file.CacheRemoteFileData(fileHash, offset, tmpFileName) == nil {
			rawData, err = file.GetSliceDataFromTmp(fileHash, tmpFileName)
			if err != nil {
				return nil, errors.Wrap(err, "failed getting slice data from tmp")
			}
		}
		metrics.UploadPerformanceLogNow(fileHash + ":RCV_GET_REMOTE_DATA:" + strconv.FormatInt(int64(offset.SliceOffsetStart), 10))
	}

	if rawData == nil {
		return nil, errors.New("Failed reading data from file")
	}

	// Encrypt slice data if required
	data := rawData
	if uploadTask.RspUploadFile.IsEncrypted {
		var err error
		data, err = encryptSliceData(rawData)
		if err != nil {
			return nil, errors.Wrap(err, "Couldn't encrypt slice data")
		}
		// write data back to the tmp file
		err = file.SaveTmpSliceData(fileHash, tmpFileName, data)
		if err != nil {
			return nil, err
		}
	}
	sliceHash := utils.CalcSliceHash(data, fileHash, slice.Slice.SliceNumber)
	tk := &UploadSliceTask{
		Data:          data,
		RspUploadFile: uploadTask.RspUploadFile,
		SliceHash:     sliceHash,
		SliceNumber:   slice.Slice.SliceNumber,
	}

	err = file.RenameTmpFile(fileHash, tmpFileName, sliceHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed renaming tmp file")
	}
	return tk, nil
}

func GetReuploadSliceTask(ctx context.Context, slice *SliceWithStatus, ppInfo *protos.PPBaseInfo, uploadTask *UploadFileTask,
	sliceDataAccessor func(fileHash string, slice *protos.SliceHashAddr) ([]byte, error)) (*UploadSliceTask, error) {
	fileHash := uploadTask.RspBackupFile.FileHash
	pp.DebugLogf(ctx, "  fileHash %s sliceNumber %v, sliceHash %s", fileHash, slice.Slice.SliceNumber, slice.Slice.SliceHash)

	rawData, err := sliceDataAccessor(fileHash, slice.Slice)
	if rawData == nil {
		return nil, errors.Wrapf(err, "Failed to find the file slice in temp folder for fileHash %s sliceNumber %v, sliceHash %s",
			fileHash, slice.Slice.SliceNumber, slice.Slice.SliceHash)
	}
	tk := &UploadSliceTask{
		Data:        rawData,
		SliceNumber: slice.Slice.SliceNumber,
	}
	return tk, nil
}

func SaveUploadFile(target *protos.ReqUploadFileSlice) error {
	return file.SaveSliceData(target.Data, target.SliceHash, target.PieceOffset.SliceOffsetStart)
}

func SaveBackuptFile(target *protos.ReqBackupFileSlice) error {
	return file.SaveSliceData(target.Data, target.SliceHash, target.PieceOffset.SliceOffsetStart)
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
