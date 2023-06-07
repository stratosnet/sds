package task

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/metrics"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/utils"
)

type TransferTask struct {
	IsReceiver         bool
	DeleteOrigin       bool
	PpInfo             *protos.PPBaseInfo
	SliceStorageInfo   *protos.SliceStorageInfo
	FileHash           string
	SliceNum           uint64
	ReceiverP2pAddress string
}

var rwmutex sync.RWMutex

var transferTaskMap = make(map[string]TransferTask)

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

func CleanTransferTask(taskId, sliceHash string) {
	rwmutex.Lock()
	delete(transferTaskMap, taskId+sliceHash)
	rwmutex.Unlock()
}

func GetTransferSliceData(taskId, sliceHash string) []byte {
	if tTask, ok := GetTransferTask(taskId, sliceHash); ok {
		data, err := file.GetSliceData(tTask.SliceStorageInfo.SliceHash)
		if err != nil {
			utils.ErrorLog("failed getting slice data", err)
		}
		return data
	}
	return nil
}

func SaveTransferData(target *protos.RspTransferDownload) error {
	if tTask, ok := GetTransferTask(target.TaskId, target.SliceHash); ok {
		err := file.SaveSliceData(target.Data, tTask.SliceStorageInfo.SliceHash, target.Offset)
		if err != nil {
			return errors.Wrap(err, "failed saving slice data")
		}
		sliceSize, err := file.GetSliceSize(tTask.SliceStorageInfo.SliceHash)
		if err != nil {
			return errors.Wrap(err, "failed getting slice size")
		}
		if target.SliceSize == uint64(sliceSize) {
			return nil
		}
	}
	return errors.New("failed getting transfer task")
}
