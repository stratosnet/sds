package task

import (
	"context"
	"sync"
	"time"

	"github.com/alex023/clock"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/metrics"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/utils"
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
	SLICE_STATUS_STARTED = iota
	SLICE_STATUS_FAILED
	SLICE_STATUS_WAITING_FOR_SP
	SLICE_STATUS_REPLACED
	SLICE_STATUS_FINISHED

	MAXSLICE              = 50 // max number of slices that can upload concurrently for a single file
	UPLOAD_TIMER_INTERVAL = 10 // seconds
	MAX_UPLOAD_RETRY      = 5
	UPLOAD_WAIT_TIMEOUT   = 60 // in seconds
	STATE_NOT_STARTED     = 0
	STATE_RUNNING         = 1
	STATE_DONE            = 2
	STATE_PAUSED          = 3
)

var (
	UploadErrNoUploadTask = errors.New("no upload task found for the file")
	UploadErrFatalError   = errors.New("upload task stops on unresolvable error")
	UploadErrMaxRetries   = errors.New("upload task stops on too many retries")
	UploadFinished        = errors.New("upload task stops on success")
	TaskTimer             = clock.NewClock()
)

// UploadFileTask represents a file upload task that is in progress
type UploadFileTask struct {
	rspUploadFile     map[int]*protos.RspUploadFile
	rspBackupFile     *protos.RspBackupStatus
	uploadType        protos.UploadType
	fileCRC           uint32
	destinations      map[string]*SlicesPerDestination
	state             int
	pause             bool
	concurrentUploads int
	fatalError        error
	retryCount        int
	mutex             sync.RWMutex
	lastTouch         time.Time
	scheduledJob      clock.Job
	helper            func(ctx context.Context, fileHash string)
}

type SlicesPerDestination struct {
	ppInfo  *protos.PPBaseInfo
	slices  []*SliceWithStatus
	started bool
}

// SliceWithStatus wraps a SliceHashAddr, and it provides extra states for upload/backup task.
type SliceWithStatus struct {
	slice    *protos.SliceHashAddr
	Error    error
	fatal    bool // Whether this error should cancel the whole file upload or not
	Status   int
	CostTime int64
}

func CreateBackupFileTask(target *protos.RspBackupStatus, fn func(ctx context.Context, fileHash string)) *UploadFileTask {
	if target == nil {
		return nil
	}

	task := &UploadFileTask{
		rspUploadFile:     nil,
		rspBackupFile:     target,
		fileCRC:           utils.CalcFileCRC32(file.GetFilePath(target.FileHash)),
		uploadType:        protos.UploadType_BACKUP,
		destinations:      make(map[string]*SlicesPerDestination),
		concurrentUploads: 0,
		retryCount:        0,
		mutex:             sync.RWMutex{},
		helper:            fn,
	}

	for _, slice := range target.Slices {
		_, ok := task.destinations[slice.PpInfo.P2PAddress]
		if !ok {
			task.destinations[slice.PpInfo.P2PAddress] = &SlicesPerDestination{
				ppInfo:  slice.PpInfo,
				started: false,
			}
		}
		sws := &SliceWithStatus{
			slice:  slice,
			Status: SLICE_STATUS_STARTED,
		}
		task.destinations[slice.PpInfo.P2PAddress].slices = append(task.destinations[slice.PpInfo.P2PAddress].slices, sws)
	}
	metrics.TaskCount.WithLabelValues("upload").Inc()
	return task
}

func CreateUploadFileTask(target *protos.RspUploadFile, fn func(ctx context.Context, fileHash string)) *UploadFileTask {
	if target == nil {
		return nil
	}

	task := &UploadFileTask{
		rspUploadFile:     make(map[int]*protos.RspUploadFile),
		rspBackupFile:     nil,
		fileCRC:           utils.CalcFileCRC32(file.GetFilePath(target.FileHash)),
		uploadType:        protos.UploadType_NEW_UPLOAD,
		destinations:      make(map[string]*SlicesPerDestination),
		concurrentUploads: 0,
		retryCount:        0,
		mutex:             sync.RWMutex{},
		lastTouch:         time.Now(),
		helper:            fn,
	}
	task.rspUploadFile[0] = target

	for _, slice := range target.Slices {
		_, ok := task.destinations[slice.PpInfo.P2PAddress]
		if !ok {
			task.destinations[slice.PpInfo.P2PAddress] = &SlicesPerDestination{
				ppInfo:  slice.PpInfo,
				started: false,
			}
		}
		sws := &SliceWithStatus{
			slice:  slice,
			Status: SLICE_STATUS_STARTED,
		}
		task.destinations[slice.PpInfo.P2PAddress].slices = append(task.destinations[slice.PpInfo.P2PAddress].slices, sws)
	}
	metrics.TaskCount.WithLabelValues("upload").Inc()
	return task
}

func (u *UploadFileTask) addNewSlice(slice *protos.SliceHashAddr) {
	slicesPerDestination := u.destinations[slice.PpInfo.P2PAddress]
	if slicesPerDestination == nil {
		slicesPerDestination = &SlicesPerDestination{
			ppInfo:  slice.PpInfo,
			started: false,
		}
		u.destinations[slice.PpInfo.P2PAddress] = slicesPerDestination
	}
	u.destinations[slice.PpInfo.P2PAddress].started = false
	slicesPerDestination.slices = append(slicesPerDestination.slices, &SliceWithStatus{
		slice:  slice,
		Status: SLICE_STATUS_STARTED,
	})
}

func (u *UploadFileTask) SignalNewDestinations(ctx context.Context) {
	fileHash := ""
	if u.rspUploadFile != nil {
		fileHash = u.rspUploadFile[0].FileHash
	}
	if u.rspBackupFile != nil {
		fileHash = u.rspBackupFile.FileHash
	}
	if u.helper != nil {
		u.helper(ctx, fileHash)
	}
}

func (u *UploadFileTask) IsFinished() bool {
	u.mutex.RLock()
	defer u.mutex.RUnlock()

	for _, destination := range u.destinations {
		for _, slice := range destination.slices {
			if slice.Status != SLICE_STATUS_FINISHED && slice.Status != SLICE_STATUS_REPLACED {
				return false
			}
		}
	}

	return true
}

func (u *UploadFileTask) IsFatal() error {
	u.mutex.RLock()
	defer u.mutex.RUnlock()

	if u.fatalError != nil {
		return u.fatalError
	}
	for _, slicesPerDestination := range u.destinations {
		for _, slice := range slicesPerDestination.slices {
			if slice.fatal {
				return slice.Error
			}
		}
	}

	return nil
}
func (u *UploadFileTask) GetLastTouch() time.Time {
	u.mutex.RLock()
	defer u.mutex.RUnlock()

	return u.lastTouch
}

func (u *UploadFileTask) Touch() {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	u.lastTouch = time.Now()
}

func (u *UploadFileTask) GetUploadFileHash() string {
	u.mutex.RLock()
	defer u.mutex.RUnlock()

	return u.rspUploadFile[0].FileHash
}

func (u *UploadFileTask) GetUploadTaskId() string {
	u.mutex.RLock()
	defer u.mutex.RUnlock()

	return u.rspUploadFile[0].TaskId
}

func (u *UploadFileTask) GetUploadType() protos.UploadType {
	u.mutex.RLock()
	defer u.mutex.RUnlock()

	return u.uploadType
}

func (u *UploadFileTask) GetUploadSpP2pAddress() string {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	return u.rspUploadFile[0].SpP2PAddress
}

func (u *UploadFileTask) SetRspUploadFile(rspUploadFile *protos.RspUploadFile) {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	u.rspUploadFile[u.retryCount] = rspUploadFile
}

// SliceFailuresToReport returns the list of slices that will require a new destination, and a boolean list of the same length indicating which slices actually failed
func (u *UploadFileTask) SliceFailuresToReport() ([]*protos.SliceHashAddr, []bool) {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	var slicesToReDownload []*protos.SliceHashAddr
	var failedSlices []bool
	for _, slicesPerDestination := range u.destinations {
		failure := false
		for _, slice := range slicesPerDestination.slices {
			if slice.Status == SLICE_STATUS_FAILED || slice.Status == SLICE_STATUS_STARTED || slice.Status == SLICE_STATUS_WAITING_FOR_SP {
				slicesToReDownload = append(slicesToReDownload, slice.slice)
				failedSlices = append(failedSlices, slice.Status == SLICE_STATUS_FAILED)
				slice.Status = SLICE_STATUS_WAITING_FOR_SP
				failure = true
			}
		}
		if failure {
			// stop the destination, and it will be re-started when failure is handled
			slicesPerDestination.started = false
		}
	}

	return slicesToReDownload, failedSlices
}

func (u *UploadFileTask) CanRetry() bool {
	return u.retryCount < MAX_UPLOAD_RETRY
}

func (u *UploadFileTask) UpdateRetryCount() {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	u.retryCount++
}

func (u *UploadFileTask) GetExcludedDestinations() []*protos.PPBaseInfo {
	u.mutex.RLock()
	defer u.mutex.RUnlock()

	var destinations []*protos.PPBaseInfo
	for _, destination := range u.destinations {
		for _, slice := range destination.slices {
			if slice.Status == SLICE_STATUS_FAILED || slice.Status == SLICE_STATUS_WAITING_FOR_SP || slice.Status == SLICE_STATUS_REPLACED {
				destinations = append(destinations, destination.ppInfo)
				break
			}
		}
	}

	return destinations
}

func (u *UploadFileTask) NextDestination() *SlicesPerDestination {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	if u.concurrentUploads >= MAXSLICE {
		return nil
	}
	for _, destination := range u.destinations {
		if !destination.started {
			destination.started = true
			u.concurrentUploads++
			return destination
		}
	}

	return nil
}

func (u *UploadFileTask) UpdateSliceDestinationsForRetry(newDestinations []*protos.SliceHashAddr) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	if u.state != STATE_NOT_STARTED {
		return
	}

	// Get original destination for each slice
	originalDestinations := make(map[uint64]string)
	for p2pAddress, destination := range u.destinations {
		for _, slice := range destination.slices {
			// this slice might have been tried before and already set to SLICE_STATUS_REPLACED
			if slice.Status != SLICE_STATUS_REPLACED {
				originalDestinations[slice.slice.SliceNumber] = p2pAddress
			}
		}
	}

	// Update slice destinations
	for _, newDestination := range newDestinations {
		originalP2pAddress, ok := originalDestinations[newDestination.SliceNumber]
		if !ok {
			continue
		}

		slicesOriginalDestination := u.destinations[originalP2pAddress]
		if slicesOriginalDestination == nil {
			continue
		}
		for _, slice := range slicesOriginalDestination.slices {
			if slice.slice.SliceNumber == newDestination.SliceNumber {
				slice.Status = SLICE_STATUS_REPLACED
				u.addNewSlice(newDestination)
				break
			}
		}
	}
}

func (u *UploadFileTask) Pause() {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	u.pause = true
}

func (u *UploadFileTask) Continue() {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	u.pause = false
}

func (u *UploadFileTask) GetState() int {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	return u.state
}

func (u *UploadFileTask) SetState(state int) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	u.state = state
}

func (u *UploadFileTask) SetUploadSliceStatus(sliceHash string, status int) error {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	for _, d := range u.destinations {
		for _, s := range d.slices {
			if s == nil {
				continue
			}
			if sliceHash == s.slice.SliceHash {
				// when the slice is not in uploading, decrease the concurrent count
				if status != SLICE_STATUS_STARTED && s.Status == SLICE_STATUS_STARTED {
					u.concurrentUploads--
				}

				if s.Status == SLICE_STATUS_REPLACED {
					continue
				}
				s.Status = status
				return nil
			}
		}
	}
	return errors.New("failed finding the slice in file upload task")
}

func (u *UploadFileTask) UploadToDestination(ctx context.Context, fn func(ctx context.Context, tk *UploadSliceTask) error) {
	// start it if it's not started
	if u.state == STATE_NOT_STARTED {
		u.state = STATE_RUNNING
		if u.concurrentUploads < MAXSLICE {
			for _, destination := range u.destinations {
				if destination == nil {
					continue
				}
				if !destination.started {
					destination.started = true
					u.concurrentUploads++
					u.uploadSlicesToDestination(ctx, destination, fn)
				}
			}
			u.state = STATE_DONE
			return
		}
	}

	if u.state == STATE_DONE && u.pause {
		u.state = STATE_PAUSED
	}
}

func (u *UploadFileTask) uploadSlicesToDestination(ctx context.Context, destination *SlicesPerDestination, fn func(ctx context.Context, tk *UploadSliceTask) error) {
	for _, slice := range destination.slices {
		if u.pause {
			u.state = STATE_PAUSED
			return
		}

		if u.fatalError != nil {
			return
		}
		// if this slice is already done, don't upload it again
		if slice.Status == SLICE_STATUS_FINISHED || slice.Status == SLICE_STATUS_REPLACED {
			continue
		}

		var uploadSliceTask *UploadSliceTask
		var err error
		switch u.uploadType {
		case protos.UploadType_NEW_UPLOAD:
			uploadSliceTask, err = CreateUploadSliceTask(ctx, slice, u)
			if err != nil {
				slice.setError(err, true)
				return
			}
			utils.DebugLogf("starting to upload slice %v for file %v", slice.slice.SliceNumber, u.rspUploadFile[0].FileHash)
			err = fn(ctx, uploadSliceTask)
			if err != nil {
				utils.ErrorLogf("Error uploading slice %v: %v", uploadSliceTask.SliceHash, err.Error())
				slice.setError(err, false)
				return
			}
		case protos.UploadType_BACKUP:
			uploadSliceTask, err = GetReuploadSliceTask(ctx, slice, destination.ppInfo, u)
			if err != nil {
				slice.setError(err, true)
				return
			}
			pp.DebugLogf(ctx, "starting to backup slice %v for file %v", slice.slice.SliceNumber, u.rspUploadFile[0].FileHash)
			err = fn(ctx, uploadSliceTask)
			if err != nil {
				slice.setError(err, false)
				return
			}
		}
	}
}

func (u *UploadFileTask) SetFatalError(err error) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	u.fatalError = err
}

func (u *UploadFileTask) SetScheduledJob(fn func()) {
	u.scheduledJob, _ = TaskTimer.AddJobRepeat(UPLOAD_TIMER_INTERVAL*time.Second, 0, fn)
}

func (s *SliceWithStatus) setError(err error, fatal bool) {
	s.Error = err
	s.fatal = fatal
	s.Status = SLICE_STATUS_FAILED
}

// UploadFileTaskMap Map of file upload tasks that are in progress.
var UploadFileTaskMap = &sync.Map{} // map[string]*UploadFileTask

// UploadProgress represents the progress for an ongoing upload
type UploadProgress struct {
	Total     int64
	HasUpload int64
}

// UploadProgressMap Map of the progress for ongoing uploads
var UploadProgressMap = &sync.Map{}                        // map[string]*UploadProgress
var UploadTaskIdMap = utils.NewAutoCleanMap(1 * time.Hour) // map[fileHash]taskId  Store the task ID (from SP) for each upload so that getFileStatus knows which taskId corresponds to a fileHash

func CreateUploadSliceTask(ctx context.Context, slice *SliceWithStatus, uploadTask *UploadFileTask) (*UploadSliceTask, error) {
	utils.DebugLogf("sliceNumber %v  offsetStart = %v  offsetEnd = %v", slice.slice.SliceNumber, slice.slice.SliceOffset.SliceOffsetStart, slice.slice.SliceOffset.SliceOffsetEnd)
	tk := &UploadSliceTask{
		RspUploadFile: uploadTask.rspUploadFile[uploadTask.retryCount],
		SliceHash:     slice.slice.SliceHash,
		SliceNumber:   slice.slice.SliceNumber,
	}
	return tk, nil
}

func GetReuploadSliceTask(ctx context.Context, slice *SliceWithStatus, ppInfo *protos.PPBaseInfo, uploadTask *UploadFileTask) (*UploadSliceTask, error) {
	fileHash := uploadTask.rspBackupFile.FileHash
	pp.DebugLogf(ctx, "  fileHash %s sliceNumber %v, sliceHash %s", fileHash, slice.slice.SliceNumber, slice.slice.SliceHash)

	rawData, err := file.GetSliceDataFromTmp(fileHash, slice.slice.SliceHash)
	if rawData == nil {
		return nil, errors.Wrapf(err, "Failed to find the file slice in temp folder for fileHash %s sliceNumber %v, sliceHash %s",
			fileHash, slice.slice.SliceNumber, slice.slice.SliceHash)
	}
	tk := &UploadSliceTask{
		SliceNumber: slice.slice.SliceNumber,
	}
	return tk, nil
}

func SaveUploadFile(target *protos.ReqUploadFileSlice) error {
	return file.SaveSliceData(target.Data, target.SliceHash, target.PieceOffset.SliceOffsetStart)
}

func SaveBackuptFile(target *protos.ReqBackupFileSlice) error {
	return file.SaveSliceData(target.Data, target.SliceHash, target.PieceOffset.SliceOffsetStart)
}

func StopRepeatedUploadTaskJob(fileHash string) {
	value, ok := UploadFileTaskMap.Load(fileHash)
	if !ok {
		utils.DebugLog("upload task for file", fileHash, "failed, can't find the task data")
		return
	}
	uploadTask := value.(*UploadFileTask)
	uploadTask.scheduledJob.Cancel()
}
