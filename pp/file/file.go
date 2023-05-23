package file

import (
	"archive/tar"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/klauspost/compress/zstd"
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
var infoMutex sync.Mutex

func GetFileInfo(filePath string) (os.FileInfo, error) {
	infoMutex.Lock()
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		infoMutex.Unlock()
		return nil, err
	}
	infoMutex.Unlock()
	return fileInfo, nil
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

func GetFileHashForVideoStream(filePath, encryptionTag string) string {
	filehash := utils.CalcFileHashForVideoStream(filePath, encryptionTag)
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

func GetFileData(filePath string, offset *protos.SliceOffset) ([]byte, error) {
	rmutex.Lock()
	fin, err := os.Open(filePath)
	if err != nil {
		rmutex.Unlock()
		return nil, errors.Wrap(err, "failed open file")
	}
	defer fin.Close()
	_, _ = fin.Seek(int64(offset.SliceOffsetStart), io.SeekStart)
	data := make([]byte, offset.SliceOffsetEnd-offset.SliceOffsetStart)
	_, err = fin.Read(data)
	if err != nil {
		rmutex.Unlock()
		return nil, errors.Wrap(err, "failed reading data")
	}
	rmutex.Unlock()

	return data, nil
}

func GetSliceDataFromTmp(fileHash, sliceHash string) ([]byte, error) {
	return GetWholeFileData(GetTmpSlicePath(fileHash, sliceHash))
}

func GetSliceData(sliceHash string) ([]byte, error) {
	slicePath, err := getSlicePath(sliceHash)
	if err != nil {
		return nil, err
	}
	return GetWholeFileData(slicePath)
}

func GetWholeFileData(filePath string) ([]byte, error) {
	rmutex.Lock()
	defer rmutex.Unlock()
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func GetSliceSize(sliceHash string) (int64, error) {
	slicePath, err := getSlicePath(sliceHash)
	if err != nil {
		return 0, errors.Wrap(err, "failed getting slice path")
	}
	info, err := GetFileInfo(slicePath)
	if err != nil {
		return 0, errors.Wrap(err, "failed getting file info")
	}
	return info.Size(), nil
}
func OpenTmpFile(fileHash, fileName string) (*os.File, error) {
	tmpFileFolderPath := getTmpFileFolderPath(fileHash)
	folderPath := filepath.Join(tmpFileFolderPath)
	exist, err := PathExists(folderPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed checking path")
	}
	if !exist {
		if err = os.MkdirAll(folderPath, os.ModePerm); err != nil {
			return nil, errors.Wrap(err, "failed creating dir")
		}
	}

	fileMg, err := os.OpenFile(GetTmpSlicePath(fileHash, fileName), os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		return nil, errors.Wrap(err, "failed opening file")
	}
	return fileMg, nil
}

func RenameTmpFile(fileHash, srcFile, dstFile string) error {
	return os.Rename(GetTmpSlicePath(fileHash, srcFile), GetTmpSlicePath(fileHash, dstFile))
}

func SaveTmpSliceData(fileHash, sliceHash string, data []byte) error {
	wmutex.Lock()
	defer wmutex.Unlock()
	fileMg, err := OpenTmpFile(fileHash, sliceHash)

	if err != nil {
		return errors.Wrap(err, "failed opening tmp file")
	}

	defer func() {
		_ = fileMg.Close()
	}()

	_, err = fileMg.Write(data)
	if err != nil {
		return errors.Wrap(err, "error saving file")
	}

	return nil
}

func SaveSliceData(data []byte, sliceHash string, offset uint64) error {
	wmutex.Lock()
	defer wmutex.Unlock()
	slicePath, err := getSlicePath(sliceHash)
	if err != nil {
		return errors.Wrap(err, "failed getting slice path")
	}
	fileMg, err := os.OpenFile(slicePath, os.O_CREATE|os.O_RDWR, 0777)
	defer func() {
		_ = fileMg.Close()
	}()
	if err != nil {
		return errors.Wrap(err, "failed opening a file")
	}
	_, err = fileMg.WriteAt(data, int64(offset))
	if err != nil {
		utils.ErrorLog("error save file")
		return errors.Wrap(err, "failed writing data")
	}
	return nil
}

func WriteFile(data []byte, offset int64, fileMg *os.File) error {
	_, err := fileMg.Seek(offset, 0)
	if err != nil {
		return errors.Wrap(err, "failed seeking in file")
	}
	_, err = fileMg.Write(data)
	if err != nil {
		return errors.Wrap(err, "failed writing to file")
	}
	return nil
}

func SaveFileData(ctx context.Context, data []byte, offset int64, sliceHash, fileName, fileHash, savePath, fileReqId string) error {

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
	if err != nil {
		return errors.Wrap(err, "failed opening file")
	}
	defer func() {
		_ = fileMg.Close()
	}()
	return WriteFile(data, offset, fileMg)
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
	slicePath, err := getSlicePath(sliceHash)
	if err != nil {
		return errors.Wrap(err, "failed getting slice path")
	}
	if err := os.Remove(slicePath); err != nil {
		return errors.Wrap(err, "failed removing slice")
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

func GetTmpSlicePath(fileHash, sliceHash string) string {
	return filepath.Join(getTmpFileFolderPath(fileHash), sliceHash)
}

func getTmpFileFolderPath(fileHash string) string {
	return filepath.Join(setting.GetRootPath(), TEMP_FOLDER, fileHash)
}

func CreateTarWithZstd(source string, target string) error {
	fi, err := os.Create(target)
	if err != nil {
		return err
	}
	// Create a new zstd writer
	zw, err := zstd.NewWriter(fi)
	if err != nil {
		return err
	}
	defer zw.Close()

	// Create a new tar writer
	tw := tar.NewWriter(zw)
	defer tw.Close()

	// Walk the directory and add each file to the tar archive
	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Get the header info for the file
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}
		// Set the header's name to the relative path within the directory
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		header.Name = relPath
		// Write the header to the tar archive
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		// If the file is a regular file, write its contents to the tar archive
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func ExtractTarWithZstd(source string, target string) error {
	// Open the zstd file for reading
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a new zstd reader
	zr, err := zstd.NewReader(file)
	if err != nil {
		return err
	}
	defer zr.Close()

	// Create a new tar reader
	tr := tar.NewReader(zr)

	// Extract each file from the tar archive
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}
		// Get the absolute path of the file to be extracted
		targetPath := filepath.Join(target, header.Name)
		// Create the file or directory
		switch header.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(targetPath, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
		case tar.TypeReg:
			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer file.Close()
			if _, err := io.Copy(file, tr); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported file type '%v' in tar archive", header.Typeflag)
		}
	}
	return nil
}
