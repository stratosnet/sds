package task

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stratosnet/sds/metrics"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"google.golang.org/protobuf/proto"
)

const LOCAL_REQID string = "local"

// DownloadTaskMap PP passway download task map   make(map[string]*DownloadTask)
var DownloadTaskMap = utils.NewAutoCleanMap(1 * time.Hour)

// DownloadSliceTaskMap resource node download slice task map
var DownloadSliceTaskMap = utils.NewAutoCleanMap(1 * time.Hour)

// DownloadFileMap P download info map  make(map[string]*protos.RspFileStorageInfo)
var DownloadFileMap = utils.NewAutoCleanMap(1 * time.Hour)

// var DownloadFileProgress = &sync.Map{}

var DownloadSpeedOfProgress = &sync.Map{}

var SliceSessionMap = &sync.Map{} // key: slice reqid, value: session id (file reqid)

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

var reCount int

type VideoCacheTask struct {
	Slices     []*protos.DownloadSliceInfo
	FileHash   string
	DownloadCh chan bool
}

// DownloadTask signal task convert sliceHash list to map
type DownloadTask struct {
	TaskId        string // file task id
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

type DownloadSliceData struct {
	Data    [][]byte
	FileCrc uint32
	RawSize uint64
}

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
		TaskId:        target.TaskId,
	}
	DownloadTaskMap.Store((target.FileHash + target.WalletAddress + target.ReqId), dTask)
	metrics.TaskCount.WithLabelValues("download").Inc()
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

func CleanDownloadTask(ctx context.Context, fileHash, sliceHash, walletAddress, fileReqId string) {
	if dlTask, ok := DownloadTaskMap.Load(fileHash + walletAddress + fileReqId); ok {

		downloadTask := dlTask.(*DownloadTask)
		delete(downloadTask.SliceInfo, sliceHash)
		pp.DebugLogf(ctx, "PP reported, clean slice task")

		if len(downloadTask.SliceInfo) > 0 {
			return
		}
		pp.DebugLog(ctx, "PP reported, clean all slice task")
		DownloadTaskMap.Delete(fileHash + walletAddress + fileReqId)
	}
}

func DeleteDownloadTask(fileHash, walletAddress, fileReqId string) {
	DownloadTaskMap.Delete(fileHash + walletAddress + fileReqId)
}

func CleanDownloadFileAndConnMap(ctx context.Context, fileHash, fileReqId string) {
	DownloadSpeedOfProgress.Delete(fileHash + fileReqId)
	if f, ok := DownloadFileMap.Load(fileHash + fileReqId); ok {
		fInfo := f.(*protos.RspFileStorageInfo)
		for _, slice := range fInfo.SliceInfo {
			DownloadSliceProgress.Delete(slice.SliceStorageInfo.SliceHash + fileReqId)
			p2pserver.GetP2pServer(ctx).DeleteConnFromCache("download#" + fileHash + slice.StoragePpInfo.P2PAddress + fileReqId)
		}
	}
	DownloadFileMap.Delete(fileHash + fileReqId)
}

func CancelDownloadTask(fileHash string) {
	file.DeleteDirectory(fileHash)
}

func GetDownloadSlice(target *protos.ReqDownloadSlice, slice *protos.DownloadSliceInfo) *DownloadSliceData {
	size, buffers, err := file.ReadSliceData(slice.SliceStorageInfo.SliceHash)
	if err != nil {
		utils.ErrorLog("Failed getting slice data ", err.Error())
		return nil
	}
	rawSize := uint64(size)

	if target.RspFileStorageInfo.EncryptionTag != "" {
		encryptedSlice := protos.EncryptedSlice{}
		var data = []byte{}

		for _, buffer := range buffers {
			data = append(data, buffer...)
		}
		err = proto.Unmarshal(data, &encryptedSlice)
		if err == nil {
			rawSize = encryptedSlice.RawSize
		}
	}
	dSlice := &DownloadSliceData{
		FileCrc: utils.CalcCRC32OfSlices(buffers),
		Data:    buffers,
		RawSize: rawSize,
	}
	return dSlice

}

func SaveDownloadFile(ctx context.Context, target *protos.RspDownloadSlice, fInfo *protos.RspFileStorageInfo) error {
	metrics.DownloadPerformanceLogNow(target.FileHash + ":RCV_SLICE_DATA:" + strconv.FormatInt(int64(target.SliceInfo.SliceOffset.SliceOffsetStart+(target.SliceNumber-1)*33554432), 10) + ":")
	defer metrics.DownloadPerformanceLogNow(target.FileHash + ":RCV_SAVE_DATA:" + strconv.FormatInt(int64(target.SliceInfo.SliceOffset.SliceOffsetStart+(target.SliceNumber-1)*33554432), 10) + ":")
	return file.SaveFileData(ctx, target.Data, int64(target.SliceInfo.SliceOffset.SliceOffsetStart), target.SliceInfo.SliceHash, fInfo.FileName, target.FileHash, fInfo.SavePath, fInfo.ReqId)
}

// checkAgain only used by local file downloading session
func checkAgain(ctx context.Context, fileHash string) {
	reCount--
	if f, ok := DownloadFileMap.Load(fileHash + LOCAL_REQID); ok {
		fInfo := f.(*protos.RspFileStorageInfo)
		fName := fInfo.FileName
		if fName == "" {
			fName = fileHash
		}
		filePath := file.GetDownloadTmpPath(fileHash, fName, fInfo.SavePath)
		if CheckFileOver(ctx, fileHash, filePath) {
			DownloadFileMap.Delete(fileHash + LOCAL_REQID)
			DownloadSpeedOfProgress.Delete(fileHash + LOCAL_REQID)
			utils.Log("————————————————————————————————————download finished————————————————————————————————————")
			DoneDownload(ctx, fileHash, fName, fInfo.SavePath)
		} else {
			if reCount > 0 {
				time.Sleep(time.Second * 2)
				checkAgain(ctx, fileHash)
			}
		}
	}
}

// DoneDownload only used by local file downloading session
func DoneDownload(ctx context.Context, fileHash, fileName, savePath string) {
	filePath := file.GetDownloadTmpPath(fileHash, fileName, savePath)
	newFilePath := filePath[:len(filePath)-4]
	err := os.Rename(filePath, newFilePath)
	if err != nil {
		pp.ErrorLog(ctx, "DoneDownload", err)
	}
	err = os.Remove(file.GetDownloadCsvPath(fileHash, fileName, savePath))
	if err != nil {
		pp.ErrorLog(ctx, "DoneDownload Remove", err)
	}
	lastPath := strings.Replace(newFilePath, fileHash+"/", "", -1)
	lastPath = addSeqNum2FileName(lastPath, 0)
	// if setting.IsWindows {
	// 	lastPath = filepath.FromSlash(lastPath)
	// }
	err = os.Rename(newFilePath, lastPath)
	if err != nil {
		pp.ErrorLog(ctx, "DoneDownload Rename", err)
	}
	rmPath := strings.Replace(newFilePath, "/"+fileName, "", -1)
	err = os.Remove(rmPath)
	if err != nil {
		pp.ErrorLog(ctx, "DoneDownload Remove", err)
	}

	metrics.DownloadPerformanceLogNow(fileHash + ":RCV_DOWNLOAD_DONE:")
	if _, ok := setting.ImageMap.Load(fileHash); ok {
		pp.DebugLog(ctx, "enter imageMap》》》》》》")
		exist := false
		exist, err = file.PathExists(setting.ImagePath)
		if err != nil {
			pp.ErrorLog(ctx, "ImageMap no", err)
		}
		if !exist {
			if err = os.MkdirAll(setting.ImagePath, os.ModePerm); err != nil {
				pp.ErrorLog(ctx, "ImageMap mk no", err)
			}
		}
		pp.DebugLog(ctx, "enter imageMap creation")
		if setting.IsWindows {
			var f, imageFile *os.File
			f, err = os.Open(lastPath)
			if err != nil {
				pp.ErrorLog(ctx, "err5>>>", err)
			}
			var img []byte
			img, err = io.ReadAll(f)
			if err != nil {
				pp.ErrorLog(ctx, "img err6>>>", err)
			}
			imageFile, err = os.OpenFile(setting.ImagePath+fileHash, os.O_CREATE|os.O_RDWR, 0600)
			if err != nil {
				pp.ErrorLog(ctx, "img err7>>>", err)
			}
			_, err = imageFile.Write(img)
			if err != nil {
				pp.ErrorLog(ctx, "img err8>>>", err)
			}
			f.Close()
			imageFile.Close()
			err = os.Remove(lastPath)
			if err != nil {
				pp.ErrorLog(ctx, "err9 Remove", err)
			}
		} else {
			err = os.Rename(lastPath, setting.ImagePath+fileHash)
			if err != nil {
				pp.ErrorLog(ctx, "ImageMap Rename", err)
			}
		}

		setting.ImageMap.Delete(fileHash)
	}

}

// CheckFileOver check finished
func CheckFileOver(ctx context.Context, fileHash, filePath string) bool {
	pp.DebugLog(ctx, "CheckFileOver")

	if s, ok := DownloadSpeedOfProgress.Load(fileHash + LOCAL_REQID); ok {
		sp := s.(*DownloadSP)
		info, err := file.GetFileInfo(filePath)
		if err != nil {
			pp.ErrorLog(ctx, "failed getting file info", err)
		}

		// TODO calculate fileHash to check if download is finished
		if info.Size() == sp.RawSize {
			pp.DebugLog(ctx, "ok!")
			return true
		}
		return false
	}
	return false
}

// CheckDownloadOver check download finished
func CheckDownloadOver(ctx context.Context, fileHash string) (bool, float32) {
	pp.DebugLog(ctx, "CheckDownloadOver")
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
				if CheckFileOver(ctx, fileHash, filePath) {
					DoneDownload(ctx, fileHash, fName, fInfo.SavePath)
					CleanDownloadFileAndConnMap(ctx, fileHash, LOCAL_REQID)
					return true, 1.0
				}
				reCount = 5
				time.Sleep(time.Second * 2)
				checkAgain(ctx, fileHash)
				return true, 1
			}
			return false, float32(sp.DownloadedSize) / float32(sp.TotalSize)
		}
		return false, 0
	}
	pp.ErrorLog(ctx, "download error, failed to find the task, request download again")
	return false, 0

}

func CheckRemoteDownloadOver(ctx context.Context, fileHash, fileReqId string) {
	key := fileHash + fileReqId
	size := file.GetRemoteFileInfo(key, fileReqId)
	utils.DebugLogf("size: %v", size)
	metrics.DownloadPerformanceLogNow(fileHash + ":RCV_RPC_DOWNLOAD_DONE:")
	file.SetRemoteFileResult(key, rpc.Result{Return: rpc.SUCCESS})
	CleanDownloadFileAndConnMap(ctx, fileHash, fileReqId)
}

func DownloadProgress(ctx context.Context, fileHash, fileReqId string, size uint64) {
	if s, ok := DownloadSpeedOfProgress.Load(fileHash + fileReqId); ok {
		sp := s.(*DownloadSP)
		sp.DownloadedSize += int64(size)
		p := float32(sp.DownloadedSize) / float32(sp.TotalSize) * 100
		pp.Logf(ctx, "downloaded：%.2f %% \n", p)
		setting.DownloadProgressMap.Store(fileHash, p)
		setting.ShowProgress(ctx, p)

		// all bytes downloaded
		if sp.DownloadedSize >= sp.TotalSize {
			if file.IsFileRpcRemote(fileHash + fileReqId) {
				CheckRemoteDownloadOver(ctx, fileHash, fileReqId)
			} else {
				go CheckDownloadOver(ctx, fileHash)
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
