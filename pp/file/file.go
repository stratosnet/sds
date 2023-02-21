package file

import (
	"context"
	"encoding/csv"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

var rmutex sync.RWMutex
var wmutex sync.RWMutex

// key(fileHash) : value(file path)
var fileMap = make(map[string]string)
var ipfsFileMap = make(map[string]string)
var infoMutex sync.Mutex

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

func GetFileSuffix(fileName string) string {
	fileSuffix := path.Ext(fileName)
	return fileSuffix
}

func GetFileHash(filePath, encryptionTag string) string {
	filehash := utils.CalcFileHash(filePath, encryptionTag)
	utils.DebugLog("filehash", filehash)
	fileMap[filehash] = filePath
	return filehash
}

func GetFilePath(hash string) string {
	return fileMap[hash]
}

func ClearFileMap(hash string) {
	delete(fileMap, hash)
}

func GetFileData(filePath string, offset *protos.SliceOffset) []byte {
	rmutex.Lock()
	fin, err := os.Open(filePath)
	if err != nil {
		rmutex.Unlock()
		return nil
	}
	defer fin.Close()
	_, _ = fin.Seek(int64(offset.SliceOffsetStart), io.SeekStart)
	data := make([]byte, offset.SliceOffsetEnd-offset.SliceOffsetStart)
	_, err = fin.Read(data)
	if err != nil {
		rmutex.Unlock()
		return nil
	}
	rmutex.Unlock()

	return data
}

func GetSliceDataFromTmp(fileHash, sliceHash string) []byte {
	return GetWholeFileData(getTmpSlicePath(fileHash, sliceHash))
}

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

func GetSliceSize(sliceHash string) int64 {
	info := GetFileInfo(getSlicePath(sliceHash))
	if info != nil {
		return info.Size()
	} else {
		return 0
	}

}

func SaveTmpSliceData(fileHash, sliceHash string, data []byte) error {
	wmutex.Lock()
	defer wmutex.Unlock()

	tmpFileFolderPath := getTmpFileFolderPath(fileHash)
	folderPath := filepath.Join(tmpFileFolderPath)
	exist, err := PathExists(folderPath)
	if err != nil {
		return err
	}
	if !exist {
		if err = os.MkdirAll(folderPath, os.ModePerm); err != nil {
			return err
		}
	}

	fileMg, err := os.OpenFile(getTmpSlicePath(fileHash, sliceHash), os.O_CREATE|os.O_RDWR, 0777)
	defer func() {
		_ = fileMg.Close()
	}()
	if err != nil {
		return errors.Wrap(err, "error initializing file")
	}

	_, err = fileMg.Write(data)
	if err != nil {
		return errors.Wrap(err, "error saving file")
	}

	return nil
}

func SaveSliceData(data []byte, sliceHash string, offset uint64) bool {
	wmutex.Lock()
	defer wmutex.Unlock()
	fileMg, err := os.OpenFile(getSlicePath(sliceHash), os.O_CREATE|os.O_RDWR, 0777)
	defer func() {
		_ = fileMg.Close()
	}()
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

func SaveFileData(ctx context.Context, data []byte, offset int64, sliceHash, fileName, fileHash, savePath, fileReqId string) bool {

	utils.DebugLog("sliceHash", sliceHash)

	if IsFileRpcRemote(fileHash + fileReqId) {
		// write to rpc
		return SaveRemoteFileData(fileHash+fileReqId, fileName, data, uint64(offset))
	}
	wmutex.Lock()
	defer wmutex.Unlock()

	if fileName == "" {
		fileName = fileHash
	}
	fileMg, err := os.OpenFile(GetDownloadTmpPath(fileHash, fileName, savePath), os.O_CREATE|os.O_RDWR, 0777)
	defer func() {
		_ = fileMg.Close()
	}()
	if err != nil {
		pp.ErrorLog(ctx, "SaveFileData err", err)
	}
	if err != nil {
		pp.ErrorLog(ctx, "error initialize file", err)
		return false
	}
	// _, err = fileMg.WriteAt(data, offset)
	_, err = fileMg.Seek(offset, 0)
	if err != nil {
		pp.ErrorLog(ctx, "error save file", err)
		return false
	}
	_, err = fileMg.Write(data)
	if err != nil {
		pp.ErrorLog(ctx, "error writing to file", err)
	}
	return true
}

func SaveDownloadProgress(ctx context.Context, sliceHash, fileName, fileHash, savePath, fileReqId string) {
	if IsFileRpcRemote(fileHash + fileReqId) {
		return
	}
	wmutex.Lock()
	csvFile, err := os.OpenFile(GetDownloadCsvPath(fileHash, fileName, savePath), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	defer func() {
		_ = csvFile.Close()
	}()
	defer wmutex.Unlock()
	if err != nil {
		pp.ErrorLog(ctx, "error open downloaded file records")
	}
	writer := csv.NewWriter(csvFile)
	line := []string{sliceHash}
	err = writer.Write(line)
	if err != nil {
		pp.ErrorLog(ctx, "download csv line ", err)
	}
	writer.Flush()
}

func RecordDownloadCSV(target *protos.RspFileStorageInfo) {
	// check if downloading, if not create new, sliceHash+startPosition
	csvFile, err := os.OpenFile(GetDownloadCsvPath(target.FileHash, target.FileName, target.SavePath), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	defer func() {
		_ = csvFile.Close()
	}()
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

func CheckFileExisting(ctx context.Context, fileHash, fileName, savePath, encryptionTag, fileReqId string) bool {
	pp.DebugLog(ctx, "CheckFileExisting: file Hash", fileHash)

	// check if the target path is remote, return false for "not match"
	if IsFileRpcRemote(fileHash + fileReqId) {
		return false
	}
	filePath := ""
	if savePath == "" {
		filePath = filepath.Join(setting.Config.DownloadPath, fileName)
	} else {
		filePath = filepath.Join(setting.Config.DownloadPath, savePath, fileName)
	}
	// if setting.IsWindows {
	//	filePath = filepath.FromSlash(filePath)
	// }
	pp.DebugLog(ctx, "filePath", filePath)
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0777)
	defer func() {
		_ = file.Close()
	}()
	if err != nil {
		pp.DebugLog(ctx, "no directory specified, thus no file slices")
		return false
	}

	hash := utils.CalcFileHash(filePath, encryptionTag)
	pp.DebugLog(ctx, "hash", hash)
	if hash == fileHash {
		pp.DebugLog(ctx, "file hash matched")
		return true
	}
	pp.DebugLog(ctx, "file hash not match")
	return false
}

func CheckSliceExisting(fileHash, fileName, sliceHash, savePath, fileReqId string) bool {
	utils.DebugLog("CheckSliceExisting sliceHash", sliceHash)

	if IsFileRpcRemote(fileHash + fileReqId) {
		return false
	}

	csvFile, err := os.OpenFile(GetDownloadCsvPath(fileHash, fileName, savePath), os.O_RDONLY, 0777)
	defer func() {
		_ = csvFile.Close()
	}()
	if err != nil {
		// 没有此文件目录，因此不存在此切片
		return false
	}
	reader := csv.NewReader(csvFile)
	hashs, err := reader.ReadAll()
	if len(hashs) == 0 || err != nil {
		return false
	}

	for _, item := range hashs {
		if len(item) > 0 {
			if item[0] == sliceHash {
				return true
			}
		}
	}

	return false
}

func DeleteSlice(sliceHash string) error {
	if err := os.Remove(getSlicePath(sliceHash)); err != nil {
		utils.ErrorLog("DeleteSlice Remove", err)
		return err
	}
	return nil
}

func DeleteDirectory(fileHash string) {
	err := os.RemoveAll(filepath.Join(setting.Config.DownloadPath, fileHash))
	if err != nil {
		utils.DebugLog("DeleteDirectory err", err)
	}

}

func DeleteTmpFileSlices(ctx context.Context, fileHash string) {
	err := os.RemoveAll(filepath.Join(setting.GetRootPath(), TEMP_FOLDER, fileHash))
	if err != nil {
		pp.DebugLog(ctx, "Delete tmp folder err", err)
	}
}

func CheckFilePathEx(fileHash, fileName, savePath string) bool {
	filePath := ""
	if savePath == "" {
		filePath = filepath.Join(setting.Config.DownloadPath, fileName)
	} else {
		filePath = filepath.Join(setting.Config.DownloadPath, savePath, fileName)
	}
	utils.DebugLog("filePath", filePath)
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0777)
	defer func() {
		_ = file.Close()
	}()
	return err == nil
}

func getTmpSlicePath(fileHash, sliceHash string) string {
	return filepath.Join(getTmpFileFolderPath(fileHash), sliceHash)
}

func getTmpFileFolderPath(fileHash string) string {
	return filepath.Join(setting.GetRootPath(), TEMP_FOLDER, fileHash)
}
