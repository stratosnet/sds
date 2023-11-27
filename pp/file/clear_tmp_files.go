package file

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alex023/clock"

	"github.com/stratosnet/framework/utils"
	"github.com/stratosnet/sds/pp/setting"
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
		clearTmpDownloadVideo(ctx)
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

func ageCache(cacheFolder string, expiration time.Duration, folderCheck, fileCheck func(name string) bool) {
	folders, err := os.ReadDir(cacheFolder)
	if err != nil {
		return
	}

	var folderToRm sync.Map
	for _, folder := range folders {
		if folder.IsDir() && folderCheck(folder.Name()) {
			folderpath := filepath.Join(cacheFolder, folder.Name())
			files, err := os.ReadDir(folderpath)
			if err != nil {
				utils.DebugLog("AgeCache: failed reading file folder in tmp:", err.Error())
				return
			}
			for _, file := range files {
				if !file.IsDir() && fileCheck(file.Name()) {
					filepath := filepath.Join(folderpath, file.Name())
					stat, err := os.Stat(filepath)
					if err != nil {
						utils.DebugLog("AgeCache: failed get file stat:", err.Error())
						return
					}

					at := accessTime(stat)
					if time.Since(time.Unix(at.Sec, at.Nsec)) > expiration {
						folderToRm.Store(folderpath, true)
					}
				}
			}
		}
	}
	folderToRm.Range(func(k, v any) bool {
		fp := k.(string)
		utils.DebugLog("AgeCache: clearing ", filepath.Base(fp))
		err = os.RemoveAll(fp)
		if err != nil {
			utils.DebugLog("Failed clearing AgeCache:", filepath.Base(fp), ", ", err.Error())
		}
		return true
	})
}

func clearTmpDownloadCaches(ctx context.Context) {
	ageCache(GetTmpDownloadPath(), DEFAULT_EXP_THRESHOLD*time.Hour*24,
		func(folderName string) bool { return folderName != "videos" },
		func(fileName string) bool { return strings.HasSuffix(fileName, ".tmp") })
}

func clearTmpDownloadVideo(ctx context.Context) {
	ageCache(filepath.Join(GetTmpDownloadPath(), setting.VideoPath), DEFAULT_EXP_THRESHOLD*time.Hour*24,
		func(string) bool { return true },
		func(string) bool { return true })
}
