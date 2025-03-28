package task

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/sds-msg/protos"
)

type VerifyTask struct {
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

var mu sync.RWMutex

var (
	verifyTaskMap = make(map[string]VerifyTask)
)

func AddVerifyTask(taskId, sliceHash string, tTask VerifyTask) {
	mu.Lock()
	verifyTaskMap[taskId+sliceHash] = tTask
	mu.Unlock()
}

func GetVerifyTask(taskId, sliceHash string) (tTask VerifyTask, ok bool) {
	mu.RLock()
	tTask, ok = verifyTaskMap[taskId+sliceHash]
	mu.RUnlock()
	return
}

func GetVerifySliceData(taskId, sliceHash string) (int64, [][]byte) {
	if tTask, ok := GetVerifyTask(taskId, sliceHash); ok {
		size, buffer, err := file.ReadSliceData(tTask.FileHash, tTask.SliceStorageInfo.SliceHash)
		if err != nil {
			utils.ErrorLog("failed getting slice data", err)
		}
		return size, buffer
	}
	return 0, nil
}

func SaveVerifyData(target *protos.RspVerifyDownload) (bool, error) {
	tTask, ok := GetVerifyTask(target.TaskId, target.SliceHash)
	if !ok {
		return false, errors.Errorf("failed getting transfer task - task_id:%v  slice_hash:%v  uploader_p2p_addess:%v", target.TaskId, target.SliceHash, target.P2PAddress)
	}
	// sum up AlreadySize and update task info
	tTask, ok = AddAlreadySizeToVerifyTask(target.TaskId, target.SliceHash, uint64(len(target.Data)))
	if !ok {
		return false, errors.New("failed to update task")
	}
	if tTask.AlreadySize < target.SliceSize { // AlreadySize < SliceSize
		return false, nil
	}
	utils.DebugLogf("whole slice received, sliceHash=%v", tTask.SliceStorageInfo.SliceHash)
	return true, nil

}

func AddAlreadySizeToVerifyTask(taskId, sliceHash string, alreadySizeDelta uint64) (tTask VerifyTask, ok bool) {
	rwmutex.Lock()
	defer rwmutex.Unlock()
	tTask, ok = verifyTaskMap[taskId+sliceHash]
	if !ok {
		return
	}
	tTask.AlreadySize += alreadySizeDelta
	tTask.LastTouchTime = time.Now().Unix()
	verifyTaskMap[taskId+sliceHash] = tTask
	return
}
