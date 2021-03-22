package file

import (
	"github.com/qsnetwork/qsds/pp/setting"
	"github.com/qsnetwork/qsds/utils"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
)

var ostype = runtime.GOOS

// GetDownloadTmpPath
func GetDownloadTmpPath(fileHash, fileName, savePath string) string {
	if savePath == "" {
		downPath := GetDownloadPath(fileHash) + "/" + fileName + ".tmp"
		// if setting.Iswindows {
		// 	downPath = filepath.FromSlash(downPath)
		// }
		return downPath
	}
	downPath := GetDownloadPath(savePath+"/"+fileHash) + "/" + fileName + ".tmp"
	// if setting.Iswindows {
	// 	downPath = filepath.FromSlash(downPath)
	// }
	return downPath

}

// GetDownloadCsvPath
func GetDownloadCsvPath(fileHash, fileName, savePath string) string {
	if savePath == "" {
		csv := GetDownloadPath(fileHash) + "/" + fileName + ".csv"
		// if setting.Iswindows {
		// 	csv = filepath.FromSlash(csv)
		// }
		return csv
	}
	csv := GetDownloadPath(savePath+"/"+fileHash) + "/" + fileName + ".csv"
	// if setting.Iswindows {
	// 	csv = filepath.FromSlash(csv)
	// }
	return csv

}

// GetDownloadPath
func GetDownloadPath(fileName string) string {
	filePath := setting.Config.DownloadPath + fileName
	// if setting.Iswindows {
	// 	filePath = filepath.FromSlash(filePath)
	// }
	exist, err := PathExists(filePath)
	if utils.CheckError(err) {
		utils.ErrorLog("exist>>>>>>>>>>>>>>>", err)
		return ""
	}
	if !exist {
		if utils.CheckError(os.MkdirAll(filePath, os.ModePerm)) {
			utils.ErrorLog("MkdirAll>>>>>>>>>>>>>>.", err)
			return ""
		}
	}
	return filePath
}
func getSlicePath(hash string) string {
	s1 := string([]rune(hash)[:1])
	s2 := string([]rune(hash)[1:2])
	path := setting.Config.StorehousePath + s1 + "/" + s2
	exist, err := PathExists(path)
	if utils.CheckError(err) {
		utils.ErrorLog(err)
		return ""
	}
	if !exist {
		if utils.CheckError(os.MkdirAll(path, os.ModePerm)) {
			utils.ErrorLog(err)
			return ""
		}
	}
	return path + "/" + hash
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

// IsFile
func IsFile(f string) int {
	fi, e := os.Stat(f)
	if e != nil {
		utils.ErrorLog("IsFile", e)
		return 0
	}
	if !fi.IsDir() {
		return 1
	}
	return 2
}

// GetAllFile
func GetAllFile(pathname string) {
	utils.DebugLog("pathname", pathname)
	rd, _ := ioutil.ReadDir(pathname)
	utils.DebugLog("rd", len(rd))
	if len(rd) == 0 {
		setting.UpChan <- pathname
	}
	for _, fi := range rd {
		if fi.IsDir() == false {
			setting.UpChan <- pathname + "/" + fi.Name()
		} else {
			GetAllFile(pathname + "/" + fi.Name())
		}
	}
}

// ESCPath
func ESCPath(param []string) string {
	os := runtime.GOOS
	newStr := ""
	if os == "linux" || os == "darwin" {
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
