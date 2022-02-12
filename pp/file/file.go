package file

import (
	"encoding/csv"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

var rmutex sync.RWMutex
var wmutex sync.RWMutex

// key(fileHash) : value(file path)
var fileMap = make(map[string]string)

var infoMutex sync.Mutex

// GetFileInfo
func GetFileInfo(filePath string) os.FileInfo {
	infoMutex.Lock()
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		infoMutex.Unlock()
		return nil
	}
	infoMutex.Unlock()
	return fileInfo
}

// GetFileSuffix
func GetFileSuffix(fileName string) string {
	fileSuffix := path.Ext(fileName)
	return fileSuffix
}

// GetFileHash
func GetFileHash(filePath, encryptionTag string) string {
	filehash := utils.CalcFileHash(filePath, encryptionTag)
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
	if err != nil {
		rmutex.Unlock()
		return nil
	}
	defer fin.Close()
	_, _ = fin.Seek(int64(offset.SliceOffsetStart), os.SEEK_SET)
	data := make([]byte, offset.SliceOffsetEnd-offset.SliceOffsetStart)
	_, err = fin.Read(data)
	if err != nil {
		rmutex.Unlock()
		return nil
	}
	rmutex.Unlock()

	return data
}

// GetSliceData
func GetSliceData(sliceHash string) []byte {
	return GetWholeFileData(getSlicePath(sliceHash))
}

func GetWholeFileData(filePath string) []byte {
	rmutex.Lock()
	defer rmutex.Unlock()
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
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
	if err != nil {
		utils.ErrorLog("error initialize file")
		return false
	}
	_, err = fileMg.WriteAt(data, int64(offset))
	if err != nil {
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
	if err != nil {
		utils.Log("SaveFileData err", err)
	}
	if err != nil {
		utils.ErrorLog("error initialize file")
		wmutex.Unlock()
		return false
	}
	// _, err = fileMg.WriteAt(data, offset)
	_, err = fileMg.Seek(offset, 0)
	if err != nil {
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
	wmutex.Lock()
	csvFile, err := os.OpenFile(GetDownloadCsvPath(fileHash, fileName, savePath), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	defer csvFile.Close()
	defer wmutex.Unlock()
	if err != nil {
		utils.ErrorLog("error open downloaded file records")
	}
	writer := csv.NewWriter(csvFile)
	line := []string{sliceHash}
	err = writer.Write(line)
	if err != nil {
		utils.ErrorLog("download csv line ", err)
	}
	writer.Flush()
}

// RecordDownloadCSV
func RecordDownloadCSV(target *protos.RspFileStorageInfo) {
	// check if downloading, if not create new, sliceHash+startPosition
	csvFile, err := os.OpenFile(GetDownloadCsvPath(target.FileHash, target.FileName, target.SavePath), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	defer csvFile.Close()
	if err != nil {
		utils.ErrorLog("error open download file records")
	}
	writer := csv.NewWriter(csvFile)
	for _, rsp := range target.SliceInfo {
		sliceHash := rsp.SliceStorageInfo.SliceHash
		offsetStatrt := int64(rsp.SliceOffset.SliceOffsetStart)
		offsetEnd := int64(rsp.SliceOffset.SliceOffsetEnd)
		for {
			if offsetStatrt+setting.MAXDATA >= offsetEnd {
				offsetString := strconv.FormatInt(offsetStatrt, 10)
				line := []string{sliceHash + offsetString}
				err = writer.Write(line)
				if err != nil {
					utils.ErrorLog("download csv line ", err)
				}
				break
			}

			offsetString := strconv.FormatInt(offsetStatrt, 10)
			line := []string{sliceHash + offsetString}
			if err = writer.Write(line); err != nil {
				utils.ErrorLog("download csv line ", err)
			}
			offsetStatrt += setting.MAXDATA
		}

	}
	writer.Flush()
}

// CheckFileExisting
func CheckFileExisting(fileHash, fileName, savePath, encryptionTag string) bool {
	utils.DebugLog("CheckFileExisting: file Hash", fileHash)
	filePath := ""
	if savePath == "" {
		filePath = filepath.Join(setting.Config.DownloadPath, fileName)
	} else {
		filePath = filepath.Join(setting.Config.DownloadPath, savePath, fileName)
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

	hash := utils.CalcFileHash(filePath, encryptionTag)
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
	if err := os.Remove(getSlicePath(sliceHash)); err != nil {
		utils.ErrorLog("DeleteSlice Remove", err)
		return err
	}
	return nil
}

// DeleteDirectory DeleteDirectory
func DeleteDirectory(fileHash string) {
	err := os.RemoveAll(filepath.Join(setting.Config.DownloadPath, fileHash))
	if err != nil {
		utils.DebugLog("DeleteDirectory err", err)
	}

}

// CheckFilePathEx
func CheckFilePathEx(fileHash, fileName, savePath string) bool {
	filePath := ""
	if savePath == "" {
		filePath = filepath.Join(setting.Config.DownloadPath, fileName)
	} else {
		filePath = filepath.Join(setting.Config.DownloadPath, savePath, fileName)
	}
	utils.DebugLog("filePath", filePath)
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0777)
	defer file.Close()
	if err != nil {
		return false
	}
	return true
}
