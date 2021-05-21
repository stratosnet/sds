package file

import (
	"encoding/csv"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"sync"
)

var rmutex sync.RWMutex
var wmutex sync.RWMutex

// key(fileHash) : value(file path)
var fileMap = make(map[string]string)

var infoMutex sync.Mutex

// GetFileInfo
func GetFileInfo(filePath string) os.FileInfo {
	infoMutex.Lock()
	fileInfo, fileInfoErr := os.Stat(filePath)
	if utils.CheckError(fileInfoErr) {
		infoMutex.Unlock()
		return nil
	}
	infoMutex.Unlock()
	return fileInfo
}

// GetFileSuffix
func GetFileSuffix(fileName string) string {
	fileSuffix := path.Ext(fileName) //获取文件后缀
	return fileSuffix
}

// GetFileHash
func GetFileHash(filePath string) string {
	filehash := utils.CalcFileHash(filePath)
	utils.DebugLog("filehash", filehash)
	fileMap[filehash] = filePath
	return filehash
}

// GetFilePath
func GetFilePath(hash string) string {
	return fileMap[hash]
}

// ClearFileMap
func ClearFileMap(hash string) {
	delete(fileMap, hash)
}

// GetFileData
func GetFileData(filePath string, offset *protos.SliceOffset) []byte {
	rmutex.Lock()
	fin, err := os.Open(filePath)
	if utils.CheckError(err) {
		rmutex.Unlock()
		return nil
	}
	defer fin.Close()
	fin.Seek(int64(offset.SliceOffsetStart), os.SEEK_SET)
	data := make([]byte, (offset.SliceOffsetEnd - offset.SliceOffsetStart))
	_, err2 := fin.Read(data)
	if utils.CheckError(err2) {
		rmutex.Unlock()
		return nil
	}
	rmutex.Unlock()

	return data
}

// GetSliceData
func GetSliceData(sliceHash string) []byte {
	rmutex.Lock()
	defer rmutex.Unlock()
	data, err := ioutil.ReadFile(getSlicePath(sliceHash))
	if utils.CheckError(err) {
		return nil
	}
	return data
}

// GetSliceSize
func GetSliceSize(sliceHash string) int64 {
	info := GetFileInfo(getSlicePath(sliceHash))
	if info != nil {
		return info.Size()
	} else {
		return 0
	}

}

// SaveSliceData
func SaveSliceData(data []byte, sliceHash string, offset uint64) bool {
	wmutex.Lock()
	defer wmutex.Unlock()
	fileMg, err := os.OpenFile(getSlicePath(sliceHash), os.O_CREATE|os.O_RDWR, 0777)
	defer fileMg.Close()
	if utils.CheckError(err) {
		utils.ErrorLog("error initialize file")
		return false
	}
	_, err2 := fileMg.WriteAt(data, int64(offset))
	if utils.CheckError(err2) {
		utils.ErrorLog("error save file")
		return false
	}
	return true
}

// SaveFileData
func SaveFileData(data []byte, offset int64, sliceHash, fileName, fileHash, savePath string) bool {

	utils.DebugLog("sliceHash", sliceHash)
	wmutex.Lock()
	if fileName == "" {
		fileName = fileHash
	}
	fileMg, err := os.OpenFile(GetDownloadTmpPath(fileHash, fileName, savePath), os.O_CREATE|os.O_RDWR, 0777)
	defer fileMg.Close()
	if utils.CheckError(err) {
		utils.Log("SaveFileData err", err)
	}
	if utils.CheckError(err) {
		utils.ErrorLog("error initialize file")
		wmutex.Unlock()
		return false
	}
	// _, err2 := fileMg.WriteAt(data, offset)
	_, err2 := fileMg.Seek(offset, 0)
	if utils.CheckError(err2) {
		utils.ErrorLog("error save file")
		wmutex.Unlock()
		return false
	}
	fileMg.Write(data)
	wmutex.Unlock()
	return true
}

// SaveDownloadProgress
func SaveDownloadProgress(sliceHash, fileName, fileHash, savePath string) {
	csvFile, err3 := os.OpenFile(GetDownloadCsvPath(fileHash, fileName, savePath), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	defer csvFile.Close()
	if utils.CheckError(err3) {
		utils.ErrorLog("error open downloaded file records")
	}
	writer := csv.NewWriter(csvFile)
	line := []string{sliceHash}
	err4 := writer.Write(line)
	if utils.CheckError(err4) {
		utils.ErrorLog("download csv line ", err4)
	}
	writer.Flush()
}

// RecordDownloadCSV
func RecordDownloadCSV(target *protos.RspFileStorageInfo) {
	// check if downloading, if not create new, sliceHash+startPosition
	csvFile, err3 := os.OpenFile(GetDownloadCsvPath(target.FileHash, target.FileName, target.SavePath), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	defer csvFile.Close()
	if utils.CheckError(err3) {
		utils.ErrorLog("error open download file records")
	}
	writer := csv.NewWriter(csvFile)
	for _, rsp := range target.SliceInfo {
		sliceHash := rsp.SliceStorageInfo.SliceHash
		offsetStatrt := int64(rsp.SliceOffset.SliceOffsetStart)
		offsetEnd := int64(rsp.SliceOffset.SliceOffsetEnd)
		for {
			if offsetStatrt+setting.MAXDATA < offsetEnd {
				offsetString := strconv.FormatInt(offsetStatrt, 10)
				line := []string{sliceHash + offsetString}
				err := writer.Write(line)
				if utils.CheckError(err) {
					utils.ErrorLog("download csv line ", err)
				}
				offsetStatrt += setting.MAXDATA
			} else {
				offsetString := strconv.FormatInt(offsetStatrt, 10)
				line := []string{sliceHash + offsetString}
				err := writer.Write(line)
				if utils.CheckError(err) {
					utils.ErrorLog("download csv line ", err)
				}
				break
			}
		}

	}
	writer.Flush()
}

// CheckFileExisting
func CheckFileExisting(fileHash, fileName, savePath string) bool {
	utils.DebugLog("CheckFileExisting: file Hash", fileHash)
	filePath := ""
	if savePath == "" {
		filePath = setting.Config.DownloadPath + fileName
	} else {
		filePath = setting.Config.DownloadPath + savePath + "/" + fileName
	}
	// if setting.IsWindows {
	// 	filePath = filepath.FromSlash(filePath)
	// }
	utils.DebugLog("filePath", filePath)
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0777)
	defer file.Close()
	if err != nil {
		utils.DebugLog("no directory specified, thus no file slices")
		return false
	}

	hash := utils.CalcFileHash(filePath)
	utils.DebugLog("hash", hash)
	if hash == fileHash {
		utils.DebugLog("file hash matched")
		return true
	}
	utils.DebugLog("file hash not match")
	return false
}

// CheckSliceExisting
func CheckSliceExisting(fileHash, fileName, sliceHash, savePath string) bool {
	utils.DebugLog("CheckSliceExisting sliceHash", sliceHash)
	csvFile, err := os.OpenFile(GetDownloadCsvPath(fileHash, fileName, savePath), os.O_RDONLY, 0777)
	defer csvFile.Close()
	if err != nil {
		// 没有此文件目录，因此不存在此切片
		return false
	}
	reader := csv.NewReader(csvFile)
	hashs, err := reader.ReadAll()
	if len(hashs) > 0 {
		for _, item := range hashs {
			if len(item) > 0 {
				if item[0] == sliceHash {
					return true
				}
			}
		}
	} else {
		return false
	}
	return false
}

// DeleteSlice
func DeleteSlice(sliceHash string) error {
	err := os.Remove(getSlicePath(sliceHash))
	if utils.CheckError(err) {
		utils.ErrorLog("DeleteSlice Remove", err)
	}
	return err
}

// DeleteDirectory DeleteDirectory
func DeleteDirectory(fileHash string) {
	err := os.RemoveAll(setting.Config.DownloadPath + fileHash)
	if utils.CheckError(err) {
		utils.DebugLog("DeleteDirectory err", err)
	}

}

// CheckFilePathEx
func CheckFilePathEx(fileHash, fileName, savePath string) bool {
	filePath := ""
	if savePath == "" {
		filePath = setting.Config.DownloadPath + fileName
	} else {
		filePath = setting.Config.DownloadPath + savePath + "/" + fileName
	}
	utils.DebugLog("filePath", filePath)
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0777)
	defer file.Close()
	if err != nil {
		return false
	}
	return true
}
