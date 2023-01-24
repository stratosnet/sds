package task

import (
	"sync"

	"github.com/stratosnet/sds/metrics"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/file"
)

type TransferTask struct {
	IsReceiver       bool
	DeleteOrigin     bool
	PpInfo           *protos.PPBaseInfo
	SliceStorageInfo *protos.SliceStorageInfo
	FileHash         string
	SliceNum         uint64
}

var rwmutex sync.RWMutex

var transferTaskMap = make(map[string]TransferTask)

// CheckTransfer check whether can transfer
// todo:
func CheckTransfer(target *protos.ReqFileSliceBackupNotice) bool {
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

func CleanTransferTask(taskId, sliceHash string) {
	rwmutex.Lock()
	delete(transferTaskMap, taskId+sliceHash)
	rwmutex.Unlock()
}

func GetTransferSliceData(taskId, sliceHash string) []byte {
	if tTask, ok := GetTransferTask(taskId, sliceHash); ok {
		data := file.GetSliceData(tTask.SliceStorageInfo.SliceHash)
		return data
	}
	return nil
}

func SaveTransferData(target *protos.RspTransferDownload) bool {
	if tTask, ok := GetTransferTask(target.TaskId, target.SliceHash); ok {
		save := file.SaveSliceData(target.Data, tTask.SliceStorageInfo.SliceHash, target.Offset)
		if save {
			if target.SliceSize == uint64(file.GetSliceSize(tTask.SliceStorageInfo.SliceHash)) {
				return true
			}
			return false
		}
		return false
	}
	return false
}
