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
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/pp/api/rpc"
	"golang.org/x/exp/mmap"

	"github.com/stratosnet/sds/framework/crypto"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/sds-msg/protos"
)

const (
	LOCAL_TAG = "LOCAL"
)

var (
	rmutex sync.RWMutex

	wmutex sync.RWMutex

	// key(fileHash) : value(file path)
	fileMap           = make(map[string]string)
	infoMutex         sync.Mutex
	DataBuffer        sync.Mutex
	fileInfoMap       = utils.NewAutoCleanMap(1 * time.Hour)
	downloadMap       = utils.NewAutoCleanMap(1 * time.Hour)
	downloadSliceChan = &sync.Map{}
	downloadShareChan = &sync.Map{}
)

type DownloadSlice struct {
	SliceHash       string
	SliceFileOffset uint64
	SliceSize       uint64
	Handled         bool
}
type DownloadFileSlice struct {
	CurrentSlice       *DownloadSlice
	CurrentSliceOffset uint64
	SlicesToDownload   []*DownloadSlice
}

type DownloadFile struct {
	FileHash string
	FileSize uint64
	FileName string
	Slices   []DownloadSlice
}

func RequestBuffersForSlice(size int64) [][]byte {
	DataBuffer.Lock()
	defer DataBuffer.Unlock()

	var buffers [][]byte
	var start int64

	for start = 0; start < size; start += setting.MaxData {
		var end int64
		if start+int64(setting.MaxData) > size {
			end = size
		} else {
			end = start + setting.MaxData
		}
		buffer := utils.RequestBuffer()[0 : end-start]
		buffers = append(buffers, buffer)
	}

	return buffers
}

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
	filehash, err := crypto.CalcFileHash(filePath, encryptionTag, crypto.SDS_CODEC)
	if err != nil {
		utils.ErrorLog(err)
	}
	utils.DebugLog("filehash", filehash)
	fileMap[filehash] = filePath
	return filehash
}

func GetFileHashForVideoStream(filePath, encryptionTag string) string {
	filehash, err := crypto.CalcFileHash(filePath, encryptionTag, crypto.VIDEO_CODEC)
	if err != nil {
		utils.ErrorLog(err)
	}
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

func ReadFileDataToPackets(r *mmap.ReaderAt, path string) (size int64, buffer [][]byte, err error) {
	size = 0
	buffer = nil
	info, err := GetFileInfo(path)
	if err != nil {
		return
	}
	size = info.Size()
	buffer = RequestBuffersForSlice(size)

	var i int64
	for i = 0; i*setting.MaxData < size; i++ {
		_, err = r.ReadAt(buffer[i], i*setting.MaxData)
		if err != nil {
			return
		}
	}
	return
}

func ReadSliceDataFromTmp(fileHash, sliceHash string) (int64, [][]byte, error) {
	slicePath := GetTmpSlicePath(fileHash, sliceHash)
	r, err := mmap.Open(slicePath)
	if err != nil {
		return 0, nil, err
	}

	return ReadFileDataToPackets(r, slicePath)
}

func GetSliceDataFromTmp(fileHash, sliceHash string) ([]byte, error) {
	return GetWholeFileData(GetTmpSlicePath(fileHash, sliceHash))
}

func ReadSliceData(fileHash, sliceHash string) (int64, [][]byte, error) {

	slicePath, err := getSlicePath(sliceHash)
	if err != nil {
		return 0, nil, err
	}
	r, err := mmap.Open(slicePath)
	if err != nil {
		slicePath = GetTmpSlicePath(fileHash, sliceHash)
		r, err = mmap.Open(slicePath)
		if err != nil {
			return 0, nil, err
		}
	}

	return ReadFileDataToPackets(r, slicePath)
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
	tmpFileFolderPath := GetTmpFileFolderPath(fileHash)
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

	fileMg, err := os.OpenFile(GetTmpSlicePath(fileHash, fileName), os.O_CREATE|os.O_RDWR, 0600)
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
	fileMg, err := os.OpenFile(slicePath, os.O_CREATE|os.O_RDWR, 0600)
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

// SaveDownloadedFileData save data of downloaded file into download temporary folder
func SaveDownloadedFileData(data []byte, offset int64, sliceHash, fileName, fileHash, savePath, fileReqId string) error {

	utils.DebugLog("sliceHash", sliceHash)

	if IsFileRpcRemote(fileHash + fileReqId) {
		// write to rpc
		return SaveRemoteFileSliceData(sliceHash+fileReqId, fileHash+fileReqId, fileName, data, uint64(offset))
	}
	wmutex.Lock()
	defer wmutex.Unlock()

	if fileName == "" {
		fileName = fileHash
	}
	tmpFilePath := GetDownloadTmpFilePath(fileHash, fileName)
	fileMg, err := os.OpenFile(tmpFilePath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		fileMg, err = CreateFolderAndReopenFile(filepath.Dir(tmpFilePath), filepath.Base(tmpFilePath))
		if err != nil {
			return errors.Wrap(err, "failed open file")
		}
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
	csvFile, err := os.OpenFile(GetDownloadTmpCsvPath(fileHash, fileName), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
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

	SetDownloadSliceResult(fileHash+fileReqId, &rpc.Result{Return: rpc.DOWNLOAD_OK})
	if err = writer.Error(); err != nil {
		pp.ErrorLog(ctx, "flush error,", err.Error())
	}
}

func RecordDownloadCSV(target *protos.RspFileStorageInfo) {
	// check if downloading, if not create new, sliceHash+startPosition
	csvFile, err := os.OpenFile(GetDownloadTmpCsvPath(target.FileHash, target.FileName), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
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
			if offsetStatrt+setting.MaxData >= offsetEnd {
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
			offsetStatrt += setting.MaxData
		}

	}
	writer.Flush()
}

func CheckFileExisting(ctx context.Context, fileHash, fileName, savePath, encryptionTag, fileReqId string) bool {
	utils.DebugLog("CheckFileExisting: file Hash", fileHash)

	// check if the target path is remote, return false for "not match"
	if IsFileRpcRemote(fileHash + fileReqId) {
		return false
	}
	filePath := GetDownloadFilePath(fileName, savePath)
	utils.DebugLog("filePath", filePath)
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0600)
	defer func() {
		_ = file.Close()
	}()
	if err != nil {
		utils.DebugLog("check file existing: file doesn't exist.")
		return false
	}

	hash, err := crypto.CalcFileHash(filePath, encryptionTag, crypto.SDS_CODEC)
	if err != nil {
		utils.ErrorLog(err)
	}
	utils.DebugLog("hash", hash)
	if hash == fileHash {
		utils.DebugLog("file hash matched")
		return true
	}
	utils.DebugLog("file hash not match")
	return false
}
func copyFile(srcPath, dstPath string) (int64, error) {
	sourceFileStat, err := os.Stat(srcPath)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", srcPath)
	}

	source, err := os.Open(srcPath)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dstPath)
	if err != nil {
		// creat the folder and retry
		destination, err = CreateFolderAndReopenFile(filepath.Dir(dstPath), filepath.Base(dstPath))
		if err != nil {
			return 0, err
		}
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func CopyDownloadFile(fileHash, fileName, savePath string) error {
	_, err := copyFile(GetDownloadTmpFilePath(fileHash, fileName), GetDownloadFilePath(fileName, ""))
	return err
}

func CheckSliceExisting(fileHash, fileName, sliceHash, fileReqId string) bool {
	utils.DebugLog("CheckSliceExisting sliceHash", sliceHash)

	if IsFileRpcRemote(fileHash + fileReqId) {
		return false
	}

	csvFile, err := os.OpenFile(GetDownloadTmpCsvPath(fileHash, fileName), os.O_RDONLY, 0600)
	defer func() {
		_ = csvFile.Close()
	}()
	if err != nil {
		// file path not available, accordingly slice not exist
		return false
	}
	reader := csv.NewReader(csvFile)
	hashes, err := reader.ReadAll()
	if len(hashes) == 0 || err != nil {
		return false
	}

	for _, item := range hashes {
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
	err := os.RemoveAll(getDownloadTmpFolderPath(fileHash))
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

func CheckFilePathEx(filePath string) bool {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0600)
	defer func() {
		_ = file.Close()
	}()
	return err == nil
}

func GetTmpSlicePath(fileHash, sliceHash string) string {
	return filepath.Join(GetTmpFileFolderPath(fileHash), sliceHash)
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

// CheckDownloadCache check there is download cache for the file with fileHash
func CheckDownloadCache(fileHash string) error {
	fileInfo, err := os.Stat(getDownloadTmpFolderPath(fileHash))
	if err != nil {
		return errors.Wrap(err, "download cache doesn't exist, ")
	}
	if !fileInfo.IsDir() {
		return errors.New("the supposed directory name is a file")
	}
	fileName, err := GetDownloadFileNameFromTmp(fileHash)
	if err != nil {
		return errors.Wrap(err, "failed get the download file name, ")
	}

	filePath := GetDownloadTmpFilePath(fileHash, fileName)
	if fileHash != GetFileHash(filePath, "") {
		return errors.New("the cached file doesn't match file hash")
	}
	return nil
}

func SetDownloadFileInfo(fileHash string, fileInfo DownloadFile) {
	fileInfoMap.Store(fileHash, fileInfo)
}

// ReadDownloadCachedData read setting.MacData bytes from the cache and store the cursor for next reading; check the end if cursor equals file size.
func ReadDownloadCachedData(fileHash, reqid string) ([]byte, uint64, uint64, bool) {
	var offsetEnd uint64
	var offsetStart uint64
	var finished bool
	offsetEnd = 0
	finished = false
	start, ok := downloadMap.Load(fileHash + reqid)
	if !ok {
		downloadMap.Store(fileHash+reqid, 0)
		offsetStart = 0
	} else {
		offsetStart = start.(uint64)
	}

	fi, ok := fileInfoMap.Load(fileHash)
	fileName := fi.(DownloadFile).FileName
	if !ok {
		return nil, offsetStart, offsetEnd, finished
	}
	filePath := GetDownloadTmpFilePath(fileHash, fileName)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, offsetStart, offsetEnd, finished
	}

	if offsetStart >= uint64(fileInfo.Size()) {
		finished = true
		return nil, offsetStart, offsetEnd, finished
	}

	if offsetStart+setting.MaxData < uint64(fileInfo.Size()) {
		offsetEnd = offsetStart + setting.MaxData
	} else {
		offsetEnd = uint64(fileInfo.Size())
	}

	offset := &protos.SliceOffset{
		SliceOffsetStart: offsetStart,
		SliceOffsetEnd:   offsetEnd,
	}

	data, err := GetFileData(filePath, offset)
	if err != nil {
		return nil, offsetStart, offsetEnd, finished
	}
	downloadMap.Store(fileHash+reqid, offsetEnd)
	return data, offsetStart, offsetEnd, finished
}

// FinishLocalDownload when a local download is done, successfully or unsuccessfully, call this to untag
func FinishLocalDownload(fileHash string) {
	downloadMap.Delete(fileHash + LOCAL_TAG)
}

// StartLocalDownload when a local download starts, call this to tag a local download is on
func StartLocalDownload(fileHash string) {
	downloadMap.Store(fileHash+LOCAL_TAG, 0)
}

// StartLocalDownload when a local download starts, call this to tag a local download is on
func IsLocalDownload(fileHash string) bool {
	_, ok := downloadMap.Load(fileHash + LOCAL_TAG)
	return ok
}

func GetFileName(fileHash string) string {
	fileInfo, ok := fileInfoMap.Load(fileHash)
	if !ok {
		return ""
	}
	return fileInfo.(DownloadFile).FileName
}

func CreateFolderAndReopenFile(folderPath, fileName string) (*os.File, error) {
	exist, err := PathExists(folderPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed checking folder existence")
	}
	if !exist {
		if err = os.MkdirAll(folderPath, os.ModePerm); err != nil {
			return nil, errors.Wrap(err, "failed creating folder")
		}
	}
	file, err := os.OpenFile(filepath.Join(folderPath, fileName), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, errors.Wrap(err, "failed open the file after second try")
	}
	return file, nil
}

func rangeCsvFile(fileHash, fileName string, f func(sliceInCsv string)) bool {
	csvFile, err := os.OpenFile(GetDownloadTmpCsvPath(fileHash, fileName), os.O_RDONLY, 0600)
	defer func() {
		_ = csvFile.Close()
	}()
	if err != nil {
		// file path not available, accordingly slice not exist
		return false
	}
	reader := csv.NewReader(csvFile)
	hashes, err := reader.ReadAll()
	if len(hashes) == 0 || err != nil {
		return false
	}
	for _, hash := range hashes {
		if len(hash) > 0 {
			f(hash[0])
		}
	}
	return false
}

func UpdateDownloadSlices(fileInfo DownloadFile, reqId string) (updated bool) {
	updated = false
	d, found := downloadMap.Load(fileInfo.FileHash + reqId)
	if !found {
		d = DownloadFileSlice{}
	}
	downloadFileSlice := d.(DownloadFileSlice)
	rangeCsvFile(fileInfo.FileHash, fileInfo.FileName, func(sliceInCsv string) {
		found = false
		for _, slice := range downloadFileSlice.SlicesToDownload {
			if sliceInCsv == slice.SliceHash {
				found = true
				break
			}
		}

		if !found {
			updated = true
			for _, slice := range fileInfo.Slices {
				if slice.SliceHash == sliceInCsv {
					newSlice := &DownloadSlice{
						SliceHash:       slice.SliceHash,
						SliceFileOffset: slice.SliceFileOffset,
						SliceSize:       slice.SliceSize,
						Handled:         false,
					}
					downloadFileSlice.SlicesToDownload = append(downloadFileSlice.SlicesToDownload, newSlice)
				}
			}
		}
	})
	downloadMap.Store(fileInfo.FileHash+reqId, downloadFileSlice)
	return
}

func WaitDownloadSlice(fileHash, reqId string) bool {
	var fileInfo DownloadFile
	found := false
	for !found {
		f, found := fileInfoMap.Load(fileHash)
		if found {
			fileInfo = f.(DownloadFile)
			break
		}
		<-SubscribeDownloadSlice(fileHash + reqId)
		UnsubscribeDownloadSlice(fileHash + reqId)
	}
	if UpdateDownloadSlices(fileInfo, reqId) {
		return true
	}
	if CheckRemoteFileDownloadComplete(fileHash, reqId, fileInfo.FileSize) {
		return false
	}
	result := <-SubscribeDownloadSlice(fileHash + reqId)
	UnsubscribeDownloadSlice(fileHash + reqId)
	if result.Return == rpc.DOWNLOAD_OK {
		UpdateDownloadSlices(fileInfo, reqId)
	}
	return true
}

func CheckRemoteFileDownloadComplete(fileHash, reqid string, fileSize uint64) bool {
	d, _ := downloadMap.Load(fileHash + reqid)
	var size uint64
	for _, slice := range d.(DownloadFileSlice).SlicesToDownload {
		if slice.Handled {
			size = size + slice.SliceSize
		}
	}
	return size == fileSize
}

func NextRemoteDownloadPacket(fileHash, reqid string) ([]byte, uint64, uint64, bool) {
	var end uint64
	var start uint64
	var finished bool

	d, found := downloadMap.Load(fileHash + reqid)
	if !found {
		d = DownloadFileSlice{}
		downloadMap.Store(fileHash+reqid, d)
	}
	dslice := d.(DownloadFileSlice)
	chooseCurrentSlice := func(d DownloadFileSlice) DownloadFileSlice {
		for _, slice := range d.SlicesToDownload {
			if !slice.Handled {

				d.CurrentSlice = slice
				d.CurrentSliceOffset = d.CurrentSlice.SliceFileOffset
				slice.Handled = true
				found = true
				break
			}
		}
		return d
	}

	if dslice.CurrentSlice == nil ||
		dslice.CurrentSliceOffset == dslice.CurrentSlice.SliceSize+dslice.CurrentSlice.SliceFileOffset {
		found = false
		dslice = chooseCurrentSlice(dslice)
		if !found {
			if !WaitDownloadSlice(fileHash, reqid) {
				return nil, 0, 0, true
			}
			d, _ = downloadMap.Load(fileHash + reqid)
			dslice = chooseCurrentSlice(d.(DownloadFileSlice))
		}
		downloadMap.Store(fileHash+reqid, dslice)
	}
	end = 0
	finished = false
	start = dslice.CurrentSliceOffset
	sliceEnd := dslice.CurrentSlice.SliceFileOffset + dslice.CurrentSlice.SliceSize
	fi, ok := fileInfoMap.Load(fileHash)
	fileName := fi.(DownloadFile).FileName
	if !ok {
		return nil, start, end, finished
	}
	filePath := GetDownloadTmpFilePath(fileHash, fileName)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, start, end, finished
	}

	if start >= uint64(fileInfo.Size()) {
		finished = true
		return nil, start, end, finished
	}

	if start+setting.MaxData < sliceEnd {
		end = start + setting.MaxData
	} else {
		end = sliceEnd
	}

	offset := &protos.SliceOffset{
		SliceOffsetStart: start,
		SliceOffsetEnd:   end,
	}

	data, err := GetFileData(filePath, offset)
	if err != nil {
		return nil, start, end, finished
	}
	dslice.CurrentSliceOffset = end
	downloadMap.Store(fileHash+reqid, dslice)

	return data, start, end, finished

}
