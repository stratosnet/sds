package task

import (
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/file"
)

type TransferTask struct {
	FromSp           bool
	DeleteOrigin     bool
	PpInfo           *protos.PPBaseInfo
	SliceStorageInfo *protos.SliceStorageInfo
	FileHash         string
	SliceNum         uint64
}

// TransferTaskMap
var TransferTaskMap = make(map[string]TransferTask)

// CheckTransfer check whether can transfer
// todo:
func CheckTransfer(target *protos.ReqFileSliceBackupNotice) bool {
	return true
}

func AddTransferTask(taskId, sliceHash string, tTask TransferTask) {
	TransferTaskMap[taskId+sliceHash] = tTask
}

func GetTransferTask(taskId, sliceHash string) (tTask TransferTask, ok bool) {
	tTask, ok = TransferTaskMap[taskId+sliceHash]
	return
}

func CleanTransferTask(taskId, sliceHash string) {
	delete(TransferTaskMap, taskId+sliceHash)
}

// GetTransferSliceData
func GetTransferSliceData(taskId, sliceHash string) []byte {
	if tTask, ok := GetTransferTask(taskId, sliceHash); ok {
		data := file.GetSliceData(tTask.SliceStorageInfo.SliceHash)
		return data
	}
	return nil
}

// SaveTransferData
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
