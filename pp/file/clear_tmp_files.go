package file

import (
	"context"
	"fmt"
	"os"
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
	clearTmpFileJob, _ = clearTmpFileClock.AddJobRepeat(time.Second*time.Duration(checkTmpFileInterval), 0, clearAllCaches(ctx))
}
func clearAllCaches(ctx context.Context) func() {
	return func() {
		clearTmpUploadedVideos(ctx)
		clearTmpSlices(ctx)
		clearTmpDownloadCaches(ctx)
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

	// check if the folder exists
	_, error := os.Stat(baseDir)
	if os.IsNotExist(error) {
		utils.DebugLog(baseDir, " does not exist")
		return
	}

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

	// check if the folder exists
	_, error := os.Stat(baseDir)
	if os.IsNotExist(error) {
		utils.DebugLogf("%v does not exist", baseDir)
		return
	}

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

func clearTmpDownloadCaches(ctx context.Context) {
	files, err := os.ReadDir(GetTmpDownloadPath())
	if err != nil {
		utils.DebugLog("Can't open folder.")
		return
	}

	for _, file := range files {
		utils.DebugLog("File:", file.Name())
		filepath := filepath.Join(GetTmpDownloadPath(), file.Name())
		fi, err := os.Stat(filepath)
		if err != nil {
			utils.DebugLog("Can't open folder, ", err.Error())
			return
		}

		if time.Since(fi.ModTime()) > DEFAULT_EXP_THRESHOLD*time.Hour*24 {
			if err = os.RemoveAll(filepath); err != nil {
				utils.DebugLog("failed removing file:", err.Error())
			}
		}
	}
}
