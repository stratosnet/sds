package task

import (
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// DownloadTaskMap PP passway download task map   make(map[string]*DownloadTask)
var DownloadTaskMap = utils.NewAutoCleanMap(5 * time.Minute)

// DownloadFileMap P download info map  make(map[string]*protos.RspFileStorageInfo)
var DownloadFileMap = utils.NewAutoCleanMap(5 * time.Minute)

// DownloadFileProgress
// var DownloadFileProgress = &sync.Map{}

// DownloadSpeedOfProgress DownloadSpeedOfProgress
var DownloadSpeedOfProgress = &sync.Map{}

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
	}
	DownloadTaskMap.Store((target.FileHash + target.WalletAddress), dTask)
}

// CleanDownloadTask
func CleanDownloadTask(fileHash, sliceHash, walletAddress string) {
	if dlTask, ok := DownloadTaskMap.Load(fileHash + walletAddress); ok {

		downloadTask := dlTask.(*DownloadTask)
		delete(downloadTask.SliceInfo, sliceHash)
		utils.DebugLog("PP reported, clean slice task")
		if len(downloadTask.SliceInfo) > 0 {
			return
		}
		utils.DebugLog("PP reported, clean all slice task")
		DownloadTaskMap.Delete(fileHash + walletAddress)
	}
}

func DeleteDownloadTask(fileHash, walletAddress string) {
	DownloadFileMap.Delete(fileHash + walletAddress)
}

// CleanDownloadFileAndConnMap
func CleanDownloadFileAndConnMap(fileHash string) {
	DownloadSpeedOfProgress.Delete(fileHash)
	if f, ok := DownloadFileMap.Load(fileHash); ok {
		fInfo := f.(*protos.RspFileStorageInfo)
		for _, slice := range fInfo.SliceInfo {
			DownloadSliceProgress.Delete(slice.SliceStorageInfo.SliceHash)
			client.DownloadConnMap.Delete(fileHash + slice.StoragePpInfo.P2PAddress)
		}
	}
	DownloadFileMap.Delete(fileHash)
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
		return file.SaveFileData(target.Data, int64(target.SliceInfo.SliceOffset.SliceOffsetStart), target.SliceInfo.SliceHash, target.SliceInfo.SliceHash, fInfo.FileHash, fInfo.SavePath)
	} else {
		utils.DebugLog("sliceHash", target.SliceInfo.SliceHash)
		return file.SaveFileData(target.Data, int64(target.SliceInfo.SliceOffset.SliceOffsetStart), target.SliceInfo.SliceHash, fInfo.FileName, target.FileHash, fInfo.SavePath)
	}
}

func checkAgain(fileHash string) {
	reCount--
	if f, ok := DownloadFileMap.Load(fileHash); ok {
		fInfo := f.(*protos.RspFileStorageInfo)
		fName := fInfo.FileName
		if fName == "" {
			fName = fileHash
		}
		filePath := file.GetDownloadTmpPath(fileHash, fName, fInfo.SavePath)
		if CheckFileOver(fileHash, filePath) {
			DownloadFileMap.Delete(fileHash)
			DownloadSpeedOfProgress.Delete(fileHash)
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

// DoneDownload
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
	if s, ok := DownloadSpeedOfProgress.Load(fileHash); ok {
		sp := s.(*DownloadSP)
		info := file.GetFileInfo(filePath)
		if info == nil {
			return false
		}
		utils.DebugLog("info", info.Size())
		utils.DebugLog("sp.RawSize", sp.RawSize)
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
	if f, ok := DownloadFileMap.Load(fileHash); ok {
		fInfo := f.(*protos.RspFileStorageInfo)
		if s, ok := DownloadSpeedOfProgress.Load(fileHash); ok {
			sp := s.(*DownloadSP)
			if sp.DownloadedSize >= sp.TotalSize {
				fName := fInfo.FileName
				if fName == "" {
					fName = fileHash
				}
				filePath := file.GetDownloadTmpPath(fileHash, fName, fInfo.SavePath)
				if CheckFileOver(fileHash, filePath) {
					DoneDownload(fileHash, fName, fInfo.SavePath)
					CleanDownloadFileAndConnMap(fileHash)
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

// DownloadProgress
func DownloadProgress(fileHash string, size uint64) {
	if s, ok := DownloadSpeedOfProgress.Load(fileHash); ok {
		sp := s.(*DownloadSP)
		sp.DownloadedSize += int64(size)
		p := float32(sp.DownloadedSize) / float32(sp.TotalSize) * 100
		utils.Logf("downloaded：%.2f %% \n", p)
		setting.DownloadProgressMap.Store(fileHash, p)
		setting.ShowProgress(p)
		if sp.DownloadedSize >= sp.TotalSize {
			go CheckDownloadOver(fileHash)
		}
	}
}
