package file

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/utils"
)

const (
	checkTmpFileInterval  = 86400 // in seconds, 1 day
	DEFAULT_EXP_THRESHOLD = 2     // 2 days (will delete if no visit since 2 days ago)
)

var (
	clearTmpFileClock = clock.NewClock()
	clearTmpFileJob   clock.Job
)

func StartClearTmpFileJob(ctx context.Context) {
	utils.Log("Starting ClearTmpFileJob......")
	clearTmpFileJob, _ = clearTmpFileClock.AddJobRepeat(time.Second*time.Duration(checkTmpFileInterval), 0, clearTmpFile(ctx))
}

func clearTmpFile(ctx context.Context) func() {
	return func() {
		clearTmpUploadedVideos(ctx)
		clearTmpSlices(ctx)
	}
}

func StopClearTmpFileJob() {
	if clearTmpFileJob != nil {
		utils.Log("Stopping ClearTmpFileJob......")
		clearTmpFileJob.Cancel()
	}
}

func clearTmpSlices(ctx context.Context) {
	baseTmpFolderPath := GetTmpFileFolderPath("")
	baseDir := filepath.Join(baseTmpFolderPath)
	excludedDir := "logs"
	exist, err := PathExists(baseDir)
	if err != nil || !exist {
		return
	}

	// Shell command to iterate and delete subfolders based on conditions
	cmdString := fmt.Sprintf(`
	find %s -mindepth 1 -maxdepth 1 -type d ! -name %s | while read -r dir; do
    total_files=$(find "$dir" -type f | wc -l)
    old_files=$(find "$dir" -type f -atime +%s | wc -l)
    if [ "$old_files" -eq "$total_files" ] && [ "$total_files" -ne 0 ]; then
        rm -rf "$dir"
    fi
	done
	`, baseDir, excludedDir, strconv.Itoa(DEFAULT_EXP_THRESHOLD))
	//pp.DebugLogf(ctx, "command to clear tmp slices: \n%s", cmdString)
	// Execute the command
	cmd := exec.Command("sh", "-c", cmdString)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to execute command: %s\n", err)
		return
	}

	// Print output (if any)
	fmt.Println(string(out))
}

func clearTmpUploadedVideos(ctx context.Context) {
	baseTmpFolderPath := GetTmpFileFolderPath(TMP_FOLDER_VIDEO)
	baseDir := filepath.Join(baseTmpFolderPath)
	cmdString := fmt.Sprintf(`
	find %s -type f -atime +%s -exec rm {} \;
	`, baseDir, strconv.Itoa(DEFAULT_EXP_THRESHOLD))
	//pp.DebugLogf(ctx, "command to clear tmp videos: \n%s", cmdString)
	// Execute the command
	cmd := exec.Command("sh", "-c", cmdString)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to execute command: %s\n", err)
		return
	}

	// Print output (if any)
	fmt.Println(string(out))
}
