package file

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp/setting"
)

// getTmpFolderPath path to the tmp file folder
func getTmpFolderPath() string {
	return filepath.Join(setting.GetRootPath(), TEMP_FOLDER)
}

// GetTmpFileFolderPath path to tmp file folder for specific file
func GetTmpFileFolderPath(fileHash string) string {
	return filepath.Join(getTmpFolderPath(), fileHash)
}

// GetTmpDownloadPath path to the download tmp file folder
func GetTmpDownloadPath() string {
	return filepath.Join(getTmpFolderPath(), "download")
}

func getDownloadTmpFolderPath(fileHash string) string {
	return filepath.Join(GetTmpDownloadPath(), fileHash)
}

// GetDownloadTmpFilePath path to the download tmp file
func GetDownloadTmpFilePath(fileHash, fileName string) string {
	return filepath.Join(getDownloadTmpFolderPath(fileHash), fileName+".tmp")
}

// GetDownloadTmpCsvPath get download CSV path
func GetDownloadTmpCsvPath(fileHash, fileName string) string {
	return filepath.Join(getDownloadTmpFolderPath(fileHash), fileName+".csv")
}

// GetDownloadFileNameFromTmp fetch the first name in download tmp folder with the filehash, and generate the file name
func GetDownloadFileNameFromTmp(fileHash string) (string, error) {
	files, err := os.ReadDir(getDownloadTmpFolderPath(fileHash))
	if err != nil {
		return "", errors.Wrap(err, "can't get download file name, ")
	}
	for _, file := range files {
		fileName := file.Name()

		if len(fileName) > 4 && fileName[len(fileName)-4:] == ".tmp" {
			return fileName[:len(fileName)-4], nil
		}
	}
	return "", errors.New("can't find cached files")
}

// GetDownloadFilePath path to a file as in download folder
func GetDownloadFilePath(fileName, savePath string) string {
	return filepath.Join(setting.Config.Home.DownloadPath, savePath, fileName)
}

func getSlicePath(hash string) (string, error) {
	if len(hash) < 10 {
		return "", errors.New("wrong size of slice hash")
	}
	s1 := string([]rune(hash)[:8])
	s2 := string([]rune(hash)[8:10])
	path := filepath.Join(setting.Config.Home.StoragePath, s1, s2)
	exist, err := PathExists(path)
	if err != nil {
		return "", errors.Wrap(err, "failed checking path")
	}
	if !exist {
		if err = os.MkdirAll(path, os.ModePerm); err != nil {
			return "", errors.Wrap(err, "failed creating dir")
		}
	}
	return filepath.Join(path, hash), nil
}

// pathExists
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// IsFile checks if the path is a file or directory
func IsFile(f string) (bool, error) {
	fi, e := os.Stat(f)
	if e != nil {
		return false, fmt.Errorf("IsFile: error open path %v ", e)
	}
	return !fi.IsDir(), nil
}

// EscapePath
func EscapePath(param []string) string {
	operatingSystem := runtime.GOOS
	newStr := ""
	if operatingSystem == "linux" || operatingSystem == "darwin" {
		for i := 0; i < len(param); i++ {
			str := param[i]
			if str != "" {
				if str[len(str)-1:] == `\` {
					str = str[0 : len(str)-1]
				}
				newStr += str
				if i != len(param)-1 {
					newStr += " "
				}
			} else {
				newStr += " "
			}
		}
		newStr = strings.Replace(newStr, `\`, "", -1)
	} else {
		// Windows
		for i := 0; i < len(param); i++ {
			str := param[i]
			newStr += str
			newStr += " "
		}
	}
	for {
		if len(newStr) == 0 {
			return ""
		}

		if newStr[len(newStr)-1:] == " " {
			newStr = newStr[0 : len(newStr)-1]
		} else {
			break
		}
	}
	for {

		if len(newStr) == 0 {
			return ""
		}
		if newStr[:1] == " " {
			newStr = newStr[1:]
		} else {
			break
		}
	}
	utils.DebugLog("newStr", newStr)
	if newStr[:1] == `"` {
		newStr = newStr[1:]
	}
	if newStr[len(newStr)-1:] == `"` {
		newStr = newStr[:len(newStr)-1]
	}
	utils.DebugLog("path ====== ", newStr)

	return newStr
}
