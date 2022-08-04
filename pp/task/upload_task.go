package task

import (
	"encoding/json"
	"math/rand"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/encryption"
	"github.com/stratosnet/sds/utils/encryption/hdkey"
)

// UploadSliceTask
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
	SLICE_STATUS_FINISHED

	MAXSLICE              = 50 // max number of slices that can upload concurrently for a single file
	UPLOAD_TIMER_INTERVAL = 10 // seconds
)

// UploadFileTask represents a file upload task that is in progress
type UploadFileTask struct {
	FileCRC  uint32
	FileHash string
	Slices   map[string]*SlicesPerDestination
	TaskID   string
	Type     protos.UploadType

	ConcurrentUploads int
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
	SliceNumber uint64
	SliceOffset *protos.SliceOffset
	Status      int
}

func CreateUploadFileTask(fileHash, taskId string, slices []*protos.SliceNumAddr) *UploadFileTask {
	task := &UploadFileTask{
		FileCRC:           utils.CalcFileCRC32(file.GetFilePath(fileHash)),
		FileHash:          fileHash,
		Slices:            make(map[string]*SlicesPerDestination),
		TaskID:            taskId,
		Type:              protos.UploadType_NEW_UPLOAD,
		ConcurrentUploads: 0,
		UpChan:            make(chan bool, MAXSLICE),
		Mutex:             sync.RWMutex{},
	}

	for _, slice := range slices {
		if _, ok := task.Slices[slice.PpInfo.P2PAddress]; !ok {
			task.Slices[slice.PpInfo.P2PAddress] = &SlicesPerDestination{
				PpInfo:  slice.PpInfo,
				Started: false,
			}
		}
		slicesPerDestination := task.Slices[slice.PpInfo.P2PAddress]
		slicesPerDestination.Slices = append(slicesPerDestination.Slices, &SliceWithStatus{
			SliceNumber: slice.SliceNumber,
			SliceOffset: slice.SliceOffset,
			Status:      SLICE_STATUS_NOT_STARTED,
		})
	}

	return task
}

func (u *UploadFileTask) IsFinished() bool {
	u.Mutex.RLock()
	defer u.Mutex.RUnlock()

	for _, slicesPerDestination := range u.Slices {
		for _, slice := range slicesPerDestination.Slices {
			if slice.Status != SLICE_STATUS_FINISHED {
				return false
			}
		}
	}

	return true
}

func (u *UploadFileTask) IsFatal() error {
	u.Mutex.RLock()
	defer u.Mutex.RUnlock()

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
func (u *UploadFileTask) SliceFailuresToReport() ([]*protos.SliceNumAddr, []bool) {
	u.Mutex.Lock()
	defer u.Mutex.Unlock()

	var slicesToReDownload []*protos.SliceNumAddr
	var failedSlices []bool
	for _, slicesPerDestination := range u.Slices {
		errorPresent := false
		for _, slice := range slicesPerDestination.Slices {
			if slice.Status == SLICE_STATUS_FAILED {
				errorPresent = true
			}
		}

		if errorPresent {
			// There was an error sending slices to this destination, so all associated failed and not started slices will receive a new destination PP
			for _, slice := range slicesPerDestination.Slices {
				if slice.Status == SLICE_STATUS_FAILED || slice.Status == SLICE_STATUS_NOT_STARTED {
					slicesToReDownload = append(slicesToReDownload, &protos.SliceNumAddr{
						SliceNumber: slice.SliceNumber,
						SliceOffset: slice.SliceOffset,
						PpInfo:      slicesPerDestination.PpInfo,
					})
					slice.Status = SLICE_STATUS_WAITING_FOR_SP
					failedSlices = append(failedSlices, slice.Status == SLICE_STATUS_FAILED)
				}
			}
		}
	}

	return slicesToReDownload, failedSlices
}

func (u *UploadFileTask) GetDestinations() []*protos.PPBaseInfo {
	u.Mutex.RLock()
	defer u.Mutex.RUnlock()

	var destinations []*protos.PPBaseInfo
	for _, destination := range u.Slices {
		destinations = append(destinations, destination.PpInfo)
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

func CleanUpConnMap(fileHash string) {
	client.UpConnMap.Range(func(k, v interface{}) bool {
		if strings.HasPrefix(k.(string), fileHash) {
			client.UpConnMap.Delete(k.(string))
		}
		return true
	})
}

func CreateUploadSliceTask(pp *protos.SliceNumAddr, fileHash, taskID, spP2pAddress string, isVideoStream, isEncrypted bool, fileCRC uint32) (*UploadSliceTask, error) {
	if isVideoStream {
		return CreateUploadSliceTaskStream(pp, fileHash, taskID, spP2pAddress, fileCRC)
	} else {
		return CreateUploadSliceTaskFile(pp, fileHash, taskID, spP2pAddress, isEncrypted, fileCRC)
	}
}

func CreateUploadSliceTaskFile(pp *protos.SliceNumAddr, fileHash, taskID, spP2pAddress string, isEncrypted bool, fileCRC uint32) (*UploadSliceTask, error) {
	utils.DebugLogf("sliceNumber %v  offsetStart = %v  offsetEnd = %v", pp.SliceNumber, pp.SliceOffset.SliceOffsetStart, pp.SliceOffset.SliceOffsetEnd)
	startOffset := pp.SliceOffset.SliceOffsetStart
	endOffset := pp.SliceOffset.SliceOffsetEnd

	var fileSize uint64
	var filePath string

	remote := file.IsFileRpcRemote(fileHash)
	if !remote {
		// in case of local file
		filePath = file.GetFilePath(fileHash)
		fileInfo := file.GetFileInfo(filePath)
		if fileInfo == nil {
			return nil, errors.New("wrong file path")
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
	if !remote {
		rawData = file.GetFileData(filePath, offset)
	} else {
		rawData = file.GetRemoteFileData(fileHash, offset)
	}

	// Encrypt slice data if required
	data := rawData
	if isEncrypted {
		var err error
		data, err = encryptSliceData(rawData)
		if err != nil {
			return nil, errors.Wrap(err, "Couldn't encrypt slice data")
		}
	}
	dataSize := uint64(len(data))
	sliceHash := utils.CalcSliceHash(data, fileHash, pp.SliceNumber)

	sl := &protos.SliceOffsetInfo{
		SliceHash: sliceHash,
		SliceOffset: &protos.SliceOffset{
			SliceOffsetStart: 0,
			SliceOffsetEnd:   dataSize,
		},
	}

	tk := &UploadSliceTask{
		TaskID:          taskID,
		FileHash:        fileHash,
		SliceNumAddr:    pp,
		SliceOffsetInfo: sl,
		FileCRC:         fileCRC,
		Data:            data,
		SliceTotalSize:  dataSize,
		SpP2pAddress:    spP2pAddress,
	}

	err := file.SaveTmpSliceData(fileHash, sliceHash, data)
	if err != nil {
		return nil, err
	}
	return tk, nil
}

func CreateUploadSliceTaskStream(pp *protos.SliceNumAddr, fileHash, taskID, spP2pAddress string, fileCRC uint32) (*UploadSliceTask, error) {
	videoFolder := file.GetVideoTmpFolder(fileHash)
	videoSliceInfo := file.HlsInfoMap[fileHash]
	var data []byte
	var sliceTotalSize uint64

	if pp.SliceNumber == 1 {
		jsonStr, _ := json.Marshal(videoSliceInfo)
		data = jsonStr
		sliceTotalSize = uint64(len(data))
	} else if pp.SliceNumber < videoSliceInfo.StartSliceNumber {
		data = file.GetDumpySliceData(fileHash, pp.SliceNumber)
		sliceTotalSize = uint64(len(data))
	} else {
		var sliceName string
		sliceName = videoSliceInfo.SliceToSegment[pp.SliceNumber]
		slicePath := videoFolder + "/" + sliceName
		if file.GetFileInfo(slicePath) == nil {
			return nil, errors.New("wrong file path")
		}
		data = file.GetWholeFileData(slicePath)
		sliceTotalSize = uint64(file.GetFileInfo(slicePath).Size())
	}

	utils.DebugLog("sliceNumber", pp.SliceNumber)

	sliceHash := utils.CalcSliceHash(data, fileHash, pp.SliceNumber)
	offset := &protos.SliceOffset{
		SliceOffsetStart: uint64(0),
		SliceOffsetEnd:   sliceTotalSize,
	}
	sl := &protos.SliceOffsetInfo{
		SliceHash:   sliceHash,
		SliceOffset: offset,
	}
	SliceNumAddr := &protos.SliceNumAddr{
		SliceNumber: pp.SliceNumber,
		SliceOffset: offset,
		PpInfo:      pp.PpInfo,
	}
	pp.SliceOffset = offset
	tk := &UploadSliceTask{
		TaskID:          taskID,
		FileHash:        fileHash,
		SliceNumAddr:    SliceNumAddr,
		SliceOffsetInfo: sl,
		FileCRC:         fileCRC,
		Data:            data,
		SliceTotalSize:  sliceTotalSize,
		SpP2pAddress:    spP2pAddress,
	}

	err := file.SaveTmpSliceData(fileHash, sliceHash, data)
	if err != nil {
		return nil, err
	}
	return tk, nil
}

func GetReuploadSliceTask(pp *protos.SliceHashAddr, fileHash, taskID, spP2pAddress string) *UploadSliceTask {
	utils.DebugLogf("  fileHash %s sliceNumber %v, sliceHash %s",
		fileHash, pp.SliceNumber, pp.SliceHash)

	rawData := file.GetSliceDataFromTmp(fileHash, pp.SliceHash)

	if rawData == nil {
		utils.ErrorLogf("Failed to find the file slice in temp folder for fileHash %s sliceNumber %v, sliceHash %s",
			fileHash, pp.SliceNumber, pp.SliceHash)
		return nil
	}

	data := rawData
	dataSize := uint64(len(data))

	sl := &protos.SliceOffsetInfo{
		SliceHash: pp.SliceHash,
		SliceOffset: &protos.SliceOffset{
			SliceOffsetStart: 0,
			SliceOffsetEnd:   dataSize,
		},
	}

	tk := &UploadSliceTask{
		TaskID:   taskID,
		FileHash: fileHash,
		SliceNumAddr: &protos.SliceNumAddr{
			SliceNumber: pp.SliceNumber,
			SliceOffset: pp.SliceOffset,
			PpInfo:      pp.PpInfo,
		},
		SliceOffsetInfo: sl,
		Data:            data,
		SliceTotalSize:  dataSize,
		SpP2pAddress:    spP2pAddress,
	}
	return tk
}

// SaveUploadFile
func SaveUploadFile(target *protos.ReqUploadFileSlice) bool {
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
