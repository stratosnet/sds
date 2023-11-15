package task

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/metrics"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/utils"
)

const TRANSFER_TASK_TIMEOUT_THRESHOLD = 180 // in seconds

type TransferTask struct {
	TaskId             string
	IsReceiver         bool
	DeleteOrigin       bool
	PpInfo             *protos.PPBaseInfo
	SliceStorageInfo   *protos.SliceStorageInfo
	FileHash           string
	SliceNum           uint64
	ReceiverP2pAddress string
	SpP2pAddress       string
	AlreadySize        uint64
	LastTouchTime      int64
}

var rwmutex sync.RWMutex

var (
	transferTaskMap = make(map[string]TransferTask)
)

// CheckTransfer check whether can transfer
// todo:
func CheckTransfer(target *protos.NoticeFileSliceBackup) bool {
	return true
}

func AddTransferTask(taskId, sliceHash string, tTask TransferTask) {
	rwmutex.Lock()
	transferTaskMap[taskId+sliceHash] = tTask
	metrics.TaskCount.WithLabelValues("transfer").Inc()
	rwmutex.Unlock()
}

func GetOngoingTransferTaskCnt() int {
	rwmutex.RLock()
	count := len(transferTaskMap)
	rwmutex.RUnlock()
	return count
}

func GetTransferTask(taskId, sliceHash string) (tTask TransferTask, ok bool) {
	rwmutex.RLock()
	tTask, ok = transferTaskMap[taskId+sliceHash]
	rwmutex.RUnlock()
	return
}

func GetTransferTaskByTaskSliceUID(taskSliceUID string) (tTask TransferTask, ok bool) {
	rwmutex.RLock()
	tTask, ok = transferTaskMap[taskSliceUID]
	rwmutex.RUnlock()
	return
}

func AddAlreadySizeToTransferTask(taskId, sliceHash string, alreadySizeDelta uint64) (tTask TransferTask, ok bool) {
	rwmutex.Lock()
	defer rwmutex.Unlock()
	tTask, ok = transferTaskMap[taskId+sliceHash]
	if !ok {
		return
	}
	tTask.AlreadySize += alreadySizeDelta
	tTask.LastTouchTime = time.Now().Unix()
	transferTaskMap[taskId+sliceHash] = tTask
	return
}

func CleanTransferTask(taskId, sliceHash string) {
	rwmutex.Lock()
	delete(transferTaskMap, taskId+sliceHash)
	rwmutex.Unlock()
}

func CleanTransferTaskByTaskSliceUID(taskSliceUID string) {
	rwmutex.Lock()
	delete(transferTaskMap, taskSliceUID)
	rwmutex.Unlock()
}

func GetTransferSliceData(taskId, sliceHash string) (int64, [][]byte) {
	if tTask, ok := GetTransferTask(taskId, sliceHash); ok {
		size, buffer, err := file.ReadSliceData(tTask.SliceStorageInfo.SliceHash)
		if err != nil {
			utils.ErrorLog("failed getting slice data", err)
		}
		return size, buffer
	}
	return 0, nil
}

func SaveTransferData(target *protos.RspTransferDownload) (bool, error) {
	tTask, ok := GetTransferTask(target.TaskId, target.SliceHash)
	if !ok {
		return false, errors.Errorf("failed getting transfer task - task_id:%v  slice_hash:%v  uploader_p2p_addess:%v", target.TaskId, target.SliceHash, target.P2PAddress)
	}
	err := file.SaveSliceData(target.Data, tTask.SliceStorageInfo.SliceHash, target.Offset)
	if err != nil {
		return false, errors.Wrap(err, "failed saving slice data")
	}
	// sum up AlreadySize and update task info
	tTask, ok = AddAlreadySizeToTransferTask(target.TaskId, target.SliceHash, uint64(len(target.Data)))
	if !ok {
		return false, errors.New("failed to update task")
	}
	sliceSize, err := file.GetSliceSize(tTask.SliceStorageInfo.SliceHash)
	if err != nil {
		return false, errors.Wrap(err, "failed getting slice size")
	}
	if uint64(sliceSize) < tTask.SliceStorageInfo.SliceSize || // check size of local file
		tTask.AlreadySize < target.SliceSize { // AlreadySize < SliceSize
		return false, nil
	}
	// check slice hash
	sliceData, err := file.GetSliceData(tTask.SliceStorageInfo.SliceHash)
	if err != nil {
		return false, errors.Wrap(err, "Failed getting slice data")
	}
	if tTask.SliceStorageInfo.SliceHash != utils.CalcSliceHash(sliceData, tTask.FileHash, tTask.SliceNum) {
		return false, errors.New("whole slice received, but slice hash doesn't match")
	}
	utils.DebugLogf("whole slice received, sliceHash=%v", tTask.SliceStorageInfo.SliceHash)
	return true, nil

}

func GetTimeoutTransfer() []string {
	rwmutex.RLock()
	defer rwmutex.RUnlock()
	taskSliceUIDs := make([]string, 0)
	for taskSliceUID, backupTask := range transferTaskMap {
		if backupTask.LastTouchTime+TRANSFER_TASK_TIMEOUT_THRESHOLD < time.Now().Unix() {
			taskSliceUIDs = append(taskSliceUIDs, taskSliceUID)
		}
	}
	return taskSliceUIDs
}
