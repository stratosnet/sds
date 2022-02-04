package file

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

//var osType = runtime.GOOS

// GetDownloadTmpPath get temporary download path
func GetDownloadTmpPath(fileHash, fileName, savePath string) string {
	if savePath == "" {
		downPath := GetDownloadPath(fileHash) + "/" + fileName + ".tmp"
		// if setting.IsWindows {
		// 	downPath = filepath.FromSlash(downPath)
		// }
		return downPath
	}
	downPath := GetDownloadPath(savePath+"/"+fileHash) + "/" + fileName + ".tmp"
	// if setting.IsWindows {
	// 	downPath = filepath.FromSlash(downPath)
	// }
	return downPath

}

// GetDownloadCsvPath get download CSV path
func GetDownloadCsvPath(fileHash, fileName, savePath string) string {
	if savePath == "" {
		csv := GetDownloadPath(fileHash) + "/" + fileName + ".csv"
		// if setting.IsWindows {
		// 	csv = filepath.FromSlash(csv)
		// }
		return csv
	}
	csv := GetDownloadPath(savePath+"/"+fileHash) + "/" + fileName + ".csv"
	// if setting.IsWindows {
	// 	csv = filepath.FromSlash(csv)
	// }
	return csv

}

// GetDownloadPath get download path
func GetDownloadPath(fileName string) string {
	filePath := filepath.Join(setting.Config.DownloadPath, fileName)
	// if setting.IsWindows {
	// 	filePath = filepath.FromSlash(filePath)
	// }
	exist, err := PathExists(filePath)
	if err != nil {
		utils.ErrorLogf("file existed: %v", err)
		return ""
	}
	if !exist {
		if err = os.MkdirAll(filePath, os.ModePerm); err != nil {
			utils.ErrorLogf("MkdirAll error: %v", err)
			return ""
		}
	}
	return filePath
}
func getSlicePath(hash string) string {
	if len(hash) < 10 {
		utils.ErrorLog("this hash is too short")
		return ""
	}
	s1 := string([]rune(hash)[:8])
	s2 := string([]rune(hash)[8:10])
	path := filepath.Join(setting.Config.StorehousePath, s1, s2)
	exist, err := PathExists(path)
	if err != nil {
		utils.ErrorLog(err)
		return ""
	}
	if !exist {
		if err = os.MkdirAll(path, os.ModePerm); err != nil {
			utils.ErrorLog(err)
			return ""
		}
	}
	return filepath.Join(path, hash)
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

// GetAllFiles get all files in directory and all sub-directory recursively
func GetAllFiles(pathname string) {
	utils.DebugLogf("pathname: %v", pathname)
	rd, _ := ioutil.ReadDir(pathname)
	utils.DebugLogf("%v files in %v", len(rd), pathname)
	if len(rd) == 0 {
		// empty folder
		setting.UpChan <- pathname
	}
	for _, fi := range rd {
		if !fi.IsDir() {
			// file found
			setting.UpChan <- pathname + "/" + fi.Name()
			continue
		}

		// check sub folder
		GetAllFiles(pathname + "/" + fi.Name())
	}
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
