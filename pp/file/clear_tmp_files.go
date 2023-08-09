package file

import (
	"context"
	"sync"
	"time"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/utils"
)

const (
	checkTmpFileInterval         = 300  // 5 mins
	DEFAULT_UPLOAD_EXP_IN_SEC    = 3600 // 1 hour
	DEFAULT_STREAMING_EXP_IN_SEC = 1800 // 30 mins
)

var (
	clrmu = &sync.Mutex{}

	taskClearTmpFileClock = clock.NewClock()
	taskClearTmpFileJob   clock.Job
	clearTmpTaskMap       = &sync.Map{} // Key: fileHash or fileName (under tmp folder), Value: timeStamp to delete
)

func StartClearTmpFileJob(ctx context.Context) {
	utils.Log("Starting ClearTmpFileJob......")
	taskClearTmpFileJob, _ = taskClearTmpFileClock.AddJobRepeat(time.Second*time.Duration(checkTmpFileInterval), 0, clearTmpFile(ctx))
}

func clearTmpFile(ctx context.Context) func() {
	return func() {
		clrmu.Lock()
		defer clrmu.Unlock()

		entriesToDel := make([]string, 0)
		clearTmpTaskMap.Range(func(key, value interface{}) bool {
			fileHashOrName := key.(string)
			expTime := value.(int64)
			utils.Logf("iterating clearTmpTaskMap, file=%v, to be expired in %d secs", fileHashOrName, expTime-time.Now().Unix())
			if expTime > time.Now().Unix() {
				return true
			}
			DeleteTmpFileSlices(ctx, fileHashOrName)
			entriesToDel = append(entriesToDel, fileHashOrName)
			return false
		})
		// clear entries
		for _, entry := range entriesToDel {
			clearTmpTaskMap.Delete(entry)
		}
		utils.Logf("%d entries got deleted", len(entriesToDel))
	}
}

// add or update timer to clear tmp file/folders
func AddClearTmpFileChecker(fileHashOrName string, expTime int64) {
	clrmu.Lock()
	defer clrmu.Unlock()
	utils.Logf("1 new checker added to clear tmp file[%v] in %d secs", fileHashOrName, expTime-time.Now().Unix())
	clearTmpTaskMap.Store(fileHashOrName, expTime)
}
