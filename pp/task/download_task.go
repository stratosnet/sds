package task

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/utils"
)

const LOCAL_REQID string = "local"

// DownloadTaskMap PP passway download task map   make(map[string]*DownloadTask)
var DownloadTaskMap = utils.NewAutoCleanMap(5 * time.Minute)

// DownloadSliceTaskMap resource node download slice task map
var DownloadSliceTaskMap = utils.NewAutoCleanMap(1 * time.Hour)

// DownloadFileMap P download info map  make(map[string]*protos.RspFileStorageInfo)
var DownloadFileMap = utils.NewAutoCleanMap(5 * time.Minute)

// DownloadFileProgress
// var DownloadFileProgress = &sync.Map{}

// DownloadSpeedOfProgress DownloadSpeedOfProgress
var DownloadSpeedOfProgress = &sync.Map{}

// key: slice reqid, value: session id (file reqid)
var SliceSessionMap = &sync.Map{}

// DownloadSP download progress
type DownloadSP struct {
	RawSize        int64
	TotalSize      int64
	DownloadedSize int64
}

// DownloadSliceProgress hash：size
var DownloadSliceProgress = &sync.Map{}

// DownloadEncryptedSlices stores the partially downloaded encrypted slices, indexed by the slice hash.
// This is used because slices can only be decrypted after being fully downloaded
var DownloadEncryptedSlices = &sync.Map{}

var VideoCacheTaskMap = &sync.Map{}

var reCount int

type VideoCacheTask struct {
	Slices     []*protos.DownloadSliceInfo
	FileHash   string
	DownloadCh chan bool
}

// DownloadTask signal task convert sliceHash list to map
type DownloadTask struct {
	WalletAddress string
	FileHash      string
	VisitCer      string
	SliceInfo     map[string]*protos.DownloadSliceInfo
	FailedSlice   map[string]bool
	SuccessSlice  map[string]bool
	FailedPPNodes map[string]*protos.PPBaseInfo
	SliceCount    int
	taskMutex     sync.RWMutex
}

func (task *DownloadTask) SetSliceSuccess(sliceHash string) {
	task.taskMutex.Lock()
	defer task.taskMutex.Unlock()

	delete(task.FailedSlice, sliceHash)
	task.SuccessSlice[sliceHash] = true
}

func (task *DownloadTask) AddFailedSlice(sliceHash string) {
	task.taskMutex.Lock()
	defer task.taskMutex.Unlock()

	if _, ok := task.SuccessSlice[sliceHash]; ok {
		return
	}

	task.FailedSlice[sliceHash] = true
	sliceInfo, ok := task.SliceInfo[sliceHash]
	if !ok {
		return
	}
	task.FailedPPNodes[sliceInfo.StoragePpInfo.P2PAddress] = sliceInfo.StoragePpInfo
}

func (task *DownloadTask) NeedRetry() (needRetry bool) {
	task.taskMutex.Lock()
	defer task.taskMutex.Unlock()
	needRetry = len(task.FailedSlice) > 0 && len(task.SuccessSlice)+len(task.FailedSlice) == task.SliceCount
	return
}

func (task *DownloadTask) RefreshTask(target *protos.RspFileStorageInfo) {
	task.taskMutex.Lock()
	defer task.taskMutex.Unlock()
	for _, dlSliceInfo := range target.SliceInfo {
		key := dlSliceInfo.SliceStorageInfo.SliceHash
		task.SliceInfo[key] = dlSliceInfo
	}
	task.FailedSlice = make(map[string]bool)
}

// DownloadSliceData
type DownloadSliceData struct {
	Data    []byte
	FileCrc uint32
	RawSize uint64
}

// AddDownloadTask
func AddDownloadTask(target *protos.RspFileStorageInfo) {
	SliceInfoMap := make(map[string]*protos.DownloadSliceInfo)
	for _, dlSliceInfo := range target.SliceInfo {
		key := dlSliceInfo.SliceStorageInfo.SliceHash
		SliceInfoMap[key] = dlSliceInfo
	}
	dTask := &DownloadTask{
		WalletAddress: target.WalletAddress,
		FileHash:      target.FileHash,
		VisitCer:      target.VisitCer,
		SliceInfo:     SliceInfoMap,
		FailedSlice:   make(map[string]bool),
		SuccessSlice:  make(map[string]bool),
		FailedPPNodes: make(map[string]*protos.PPBaseInfo),
		SliceCount:    len(target.SliceInfo),
	}
	DownloadTaskMap.Store((target.FileHash + target.WalletAddress + target.ReqId), dTask)
}

func GetDownloadTaskWithSliceReqId(fileHash, walletAddress, sliceReqId string) (*DownloadTask, bool) {
	sid, ok := SliceSessionMap.Load(sliceReqId)
	if !ok {
		utils.DebugLog("Can't find who created slice request", sliceReqId)
		return nil, false
	}
	
	task, ok := DownloadTaskMap.Load(fileHash + walletAddress + sid.(string))
	if !ok {
		return nil, false
	}
	dTask, ok := task.(*DownloadTask)
	if !ok {
		utils.ErrorLog("failed to parse the download task for the file ", fileHash)
		return nil, false
	}
	return dTask, true
}

func GetDownloadTask(fileHash, walletAddress, fileReqId string) (*DownloadTask, bool) {
	task, ok := DownloadTaskMap.Load(fileHash + walletAddress + fileReqId)
	if !ok {
		return nil, false
	}
	dTask, ok := task.(*DownloadTask)
	if !ok {
		utils.ErrorLog("failed to parse the download task for the file ", fileHash)
		return nil, false
	}
	return dTask, true
}

func CheckDownloadTask(fileHash, walletAddress, fileReqId string) bool {
	return DownloadTaskMap.HashKey(fileHash + walletAddress + fileReqId)
}

// CleanDownloadTask
func CleanDownloadTask(fileHash, sliceHash, walletAddress, fileReqId string) {
	if dlTask, ok := DownloadTaskMap.Load(fileHash + walletAddress + fileReqId); ok {

		downloadTask := dlTask.(*DownloadTask)
		delete(downloadTask.SliceInfo, sliceHash)
		utils.DebugLogf("PP reported, clean slice task")

		if len(downloadTask.SliceInfo) > 0 {
			return
		}
		utils.DebugLog("PP reported, clean all slice task")
		DownloadTaskMap.Delete(fileHash + walletAddress + fileReqId)
	}
}

func DeleteDownloadTask(fileHash, walletAddress, fileReqId string) {
	DownloadTaskMap.Delete(fileHash + walletAddress + fileReqId)
}

// CleanDownloadFileAndConnMap
func CleanDownloadFileAndConnMap(fileHash, fileReqId string) {
	DownloadSpeedOfProgress.Delete(fileHash + fileReqId)
	if f, ok := DownloadFileMap.Load(fileHash + fileReqId); ok {
		fInfo := f.(*protos.RspFileStorageInfo)
		for _, slice := range fInfo.SliceInfo {
			DownloadSliceProgress.Delete(slice.SliceStorageInfo.SliceHash + fileReqId)
			client.DownloadConnMap.Delete(fileHash + slice.StoragePpInfo.P2PAddress + fileReqId)
		}
	}
	DownloadFileMap.Delete(fileHash + fileReqId)
}

// CancelDownloadTask
func CancelDownloadTask(fileHash string) {
	file.DeleteDirectory(fileHash)
}

// GetDownloadSlice
func GetDownloadSlice(target *protos.ReqDownloadSlice) *DownloadSliceData {
	data := file.GetSliceData(target.SliceInfo.SliceHash)
	rawSize := uint64(len(data))
	if target.IsEncrypted {
		encryptedSlice := protos.EncryptedSlice{}
		err := proto.Unmarshal(data, &encryptedSlice)
		if err == nil {
			rawSize = encryptedSlice.RawSize
		} else {
			utils.ErrorLog("Couldn't unmarshal encrypted slice to protobuf", err)
			data = []byte{}
		}
	}
	dSlice := &DownloadSliceData{
		FileCrc: utils.CalcCRC32(data),
		Data:    data,
		RawSize: rawSize,
	}
	return dSlice

}

// SaveDownloadFile
func SaveDownloadFile(target *protos.RspDownloadSlice, fInfo *protos.RspFileStorageInfo) bool {
	if fInfo.IsVideoStream {
		return file.SaveFileData(target.Data, int64(target.SliceInfo.SliceOffset.SliceOffsetStart), target.SliceInfo.SliceHash, target.SliceInfo.SliceHash, fInfo.FileHash, fInfo.SavePath, fInfo.ReqId)
	} else {
		return file.SaveFileData(target.Data, int64(target.SliceInfo.SliceOffset.SliceOffsetStart), target.SliceInfo.SliceHash, fInfo.FileName, target.FileHash, fInfo.SavePath, fInfo.ReqId)
	}
}

// checkAgain only used by local file downloading session
func checkAgain(fileHash string) {
	reCount--
	if f, ok := DownloadFileMap.Load(fileHash + LOCAL_REQID); ok {
		fInfo := f.(*protos.RspFileStorageInfo)
		fName := fInfo.FileName
		if fName == "" {
			fName = fileHash
		}
		filePath := file.GetDownloadTmpPath(fileHash, fName, fInfo.SavePath)
		if CheckFileOver(fileHash, filePath) {
			DownloadFileMap.Delete(fileHash + LOCAL_REQID)
			DownloadSpeedOfProgress.Delete(fileHash + LOCAL_REQID)
			utils.Log("————————————————————————————————————download finished————————————————————————————————————")
			DoneDownload(fileHash, fName, fInfo.SavePath)
		} else {
			if reCount > 0 {
				time.Sleep(time.Second * 2)
				checkAgain(fileHash)
			}
		}
	}
}

// DoneDownload only used by local file downloading session
func DoneDownload(fileHash, fileName, savePath string) {
	filePath := file.GetDownloadTmpPath(fileHash, fileName, savePath)
	newFilePath := filePath[:len(filePath)-4]
	err := os.Rename(filePath, newFilePath)
	if err != nil {
		utils.ErrorLog("DoneDownload", err)
	}
	err = os.Remove(file.GetDownloadCsvPath(fileHash, fileName, savePath))
	if err != nil {
		utils.ErrorLog("DoneDownload Remove", err)
	}
	lastPath := strings.Replace(newFilePath, fileHash+"/", "", -1)
	lastPath = addSeqNum2FileName(lastPath, 0)
	// if setting.IsWindows {
	// 	lastPath = filepath.FromSlash(lastPath)
	// }
	err = os.Rename(newFilePath, lastPath)
	if err != nil {
		utils.ErrorLog("DoneDownload Rename", err)
	}
	rmPath := strings.Replace(newFilePath, "/"+fileName, "", -1)
	err = os.Remove(rmPath)
	if err != nil {
		utils.ErrorLog("DoneDownload Remove", err)
	}

	if _, ok := setting.ImageMap.Load(fileHash); ok {
		utils.DebugLog("enter imageMap》》》》》》")
		exist := false
		exist, err = file.PathExists(setting.IMAGEPATH)
		if err != nil {
			utils.ErrorLog("ImageMap no", err)
		}
		if !exist {
			if err = os.MkdirAll(setting.IMAGEPATH, os.ModePerm); err != nil {
				utils.ErrorLog("ImageMap mk no", err)
			}
		}
		utils.DebugLog("enter imageMap creation")
		if setting.IsWindows {
			var f, imageFile *os.File
			f, err = os.Open(lastPath)
			if err != nil {
				utils.ErrorLog("err5>>>", err)
			}
			var img []byte
			img, err = ioutil.ReadAll(f)
			if err != nil {
				utils.ErrorLog("img err6>>>", err)
			}
			imageFile, err = os.OpenFile(setting.IMAGEPATH+fileHash, os.O_CREATE|os.O_RDWR, 0777)
			if err != nil {
				utils.ErrorLog("img err7>>>", err)
			}
			_, err = imageFile.Write(img)
			if err != nil {
				utils.ErrorLog("img err8>>>", err)
			}
			f.Close()
			imageFile.Close()
			err = os.Remove(lastPath)
			if err != nil {
				utils.ErrorLog("err9 Remove", err)
			}
		} else {
			err = os.Rename(lastPath, setting.IMAGEPATH+fileHash)
			if err != nil {
				utils.ErrorLog("ImageMap Rename", err)
			}
		}

		setting.ImageMap.Delete(fileHash)
	}

}

// CheckFileOver check finished
func CheckFileOver(fileHash, filePath string) bool {
	utils.DebugLog("CheckFileOver")

	if s, ok := DownloadSpeedOfProgress.Load(fileHash + LOCAL_REQID); ok {
		sp := s.(*DownloadSP)
		info := file.GetFileInfo(filePath)
		if info == nil {
			return false
		}

		// TODO calculate fileHash to check if download is finished
		if info.Size() == sp.RawSize {
			utils.DebugLog("ok!")
			return true
		}
		return false
	}
	return false
}

// CheckDownloadOver check download finished
func CheckDownloadOver(fileHash string) (bool, float32) {
	utils.DebugLog("CheckDownloadOver")
	if f, ok := DownloadFileMap.Load(fileHash + LOCAL_REQID); ok {
		fInfo := f.(*protos.RspFileStorageInfo)
		if s, ok := DownloadSpeedOfProgress.Load(fileHash + LOCAL_REQID); ok {
			sp := s.(*DownloadSP)
			if sp.DownloadedSize >= sp.TotalSize {
				fName := fInfo.FileName
				if fName == "" {
					fName = fileHash
				}
				filePath := file.GetDownloadTmpPath(fileHash, fName, fInfo.SavePath)
				if CheckFileOver(fileHash, filePath) {
					DoneDownload(fileHash, fName, fInfo.SavePath)
					CleanDownloadFileAndConnMap(fileHash, LOCAL_REQID)
					return true, 1.0
				}
				reCount = 5
				time.Sleep(time.Second * 2)
				checkAgain(fileHash)
				return true, 1
			}
			return false, float32(sp.DownloadedSize) / float32(sp.TotalSize)
		}
		return false, 0
	}
	utils.ErrorLog("download error, failed to find the task, request download again")
	return false, 0

}

func CheckRemoteDownloadOver(fileHash, fileReqId string) {

	key := fileHash + fileReqId
	size := file.GetRemoteFileInfo(key)
	utils.DebugLog("size:", string(size))
	file.SetRemoteFileResult(key, rpc.Result{Return:rpc.SUCCESS})
	CleanDownloadFileAndConnMap(fileHash, fileReqId)
}

// DownloadProgress
func DownloadProgress(fileHash, fileReqId string, size uint64) {
	if s, ok := DownloadSpeedOfProgress.Load(fileHash + fileReqId); ok {
		sp := s.(*DownloadSP)
		sp.DownloadedSize += int64(size)
		p := float32(sp.DownloadedSize) / float32(sp.TotalSize) * 100
		utils.Logf("downloaded：%.2f %% \n", p)
		setting.DownloadProgressMap.Store(fileHash, p)
		setting.ShowProgress(p)

		// all bytes downloaded
		if sp.DownloadedSize >= sp.TotalSize {
			if file.IsFileRpcRemote(fileHash + fileReqId) {
				CheckRemoteDownloadOver(fileHash, fileReqId)
			} else {
				go CheckDownloadOver(fileHash)
			}
		}
	}
}

func addSeqNum2FileName(filePath string, seq int) string {
	lastPath := filePath
	if seq > 0 {
		ext := filepath.Ext(filePath)
		filename := strings.TrimSuffix(filepath.Base(filePath), ext)
		if seq < 3000 {
			lastPath = fmt.Sprintf("%s/%s(%d)%s", filepath.Dir(filePath), filename, seq, ext)
		} else {
			utils.ErrorLog("Maximum sequence number of duplicate file name has been reached, use UUID instead")
			return fmt.Sprintf("%s/%s(%s)%s", filepath.Dir(filePath), filename, uuid.New().String(), ext)
		}
	}

	if exist, err := file.PathExists(lastPath); err != nil || !exist {
		return lastPath
	}

	return addSeqNum2FileName(filePath, seq+1)
}
