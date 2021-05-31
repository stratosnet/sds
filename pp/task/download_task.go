package task

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// DownloadTaskMap PP passway download task map   make(map[string]*DonwloadTask)
var DownloadTaskMap = &sync.Map{}

// DownloadFileMap P download info map  make(map[string]*protos.RspFileStorageInfo)
var DownloadFileMap = &sync.Map{}

// DonwloadFileProgress
// var DonwloadFileProgress = &sync.Map{}

// DownloadSpeedOfProgress DownloadSpeedOfProgress
var DownloadSpeedOfProgress = &sync.Map{}

// DownloadSP download progress
type DownloadSP struct {
	TotalSize    int64
	DownloadSize int64
}

// DonwloadSliceProgress hash：size
var DonwloadSliceProgress = &sync.Map{}

var reCount int

// DonwloadTask singal task convert sliceHash list to map
type DonwloadTask struct {
	WalletAddress string
	FileHash      string
	VisitCer      string
	SliceInfo     map[string]*protos.DownloadSliceInfo
}

// AddDonwloadTask
func AddDonwloadTask(target *protos.RspFileStorageInfo) {
	SliceInfoMap := make(map[string]*protos.DownloadSliceInfo)
	for _, dlSliceInfo := range target.SliceInfo {
		key := dlSliceInfo.SliceStorageInfo.SliceHash
		SliceInfoMap[key] = dlSliceInfo
	}
	dTask := &DonwloadTask{
		WalletAddress: target.WalletAddress,
		FileHash:      target.FileHash,
		VisitCer:      target.VisitCer,
		SliceInfo:     SliceInfoMap,
	}
	DownloadTaskMap.Store((target.FileHash + target.WalletAddress), dTask)
}

// CleanDownloadTask
func CleanDownloadTask(fileHash, sliceHash, wAddress string) {
	if dlTask, ok := DownloadTaskMap.Load(fileHash + wAddress); ok {

		donwloadTask := dlTask.(*DonwloadTask)
		delete(donwloadTask.SliceInfo, sliceHash)
		if len(donwloadTask.SliceInfo) == 0 {
			DownloadTaskMap.Delete((fileHash + wAddress))
			utils.DebugLog("PP reported, clean all slice task")
			client.DownloadConnMap.Delete(wAddress + fileHash)
		}
		utils.DebugLog("PP reported, clean slice taks")
	}
}

// PCleanDownloadTask p
func PCleanDownloadTask(fileHash string) {
	DownloadSpeedOfProgress.Delete(fileHash)
	if f, ok := DownloadFileMap.Load(fileHash); ok {
		fInfo := f.(*protos.RspFileStorageInfo)
		for _, slice := range fInfo.SliceInfo {
			DonwloadSliceProgress.Delete(slice.SliceStorageInfo.SliceHash)
		}
	}
	DownloadFileMap.Delete(fileHash)
}

// PCancelDownloadTask p
func PCancelDownloadTask(fileHash string) {
	file.DeleteDirectory(fileHash)
}

// DonwloadSliceData
type DonwloadSliceData struct {
	Data    []byte
	FileCrc uint32
}

// GetDonwloadSlice
func GetDonwloadSlice(target *protos.ReqDownloadSlice) *DonwloadSliceData {
	data := file.GetSliceData(target.SliceInfo.SliceHash)
	dSlice := &DonwloadSliceData{
		FileCrc: utils.CalcCRC32(data),
		Data:    data,
	}
	return dSlice

}

// SaveDownloadFile
func SaveDownloadFile(target *protos.RspDownloadSlice) bool {
	if f, ok := DownloadFileMap.Load(target.FileHash); ok {
		fInfo := f.(*protos.RspFileStorageInfo)

		return file.SaveFileData(target.Data, int64(target.SliceInfo.SliceOffset.SliceOffsetStart), target.SliceInfo.SliceHash, fInfo.FileName, target.FileHash, fInfo.SavePath)
	}
	return file.SaveFileData(target.Data, int64(target.SliceInfo.SliceOffset.SliceOffsetStart), target.SliceInfo.SliceHash, target.FileHash, target.FileHash, "")

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
			fmt.Println("————————————————————————————————————————————————————")
			fmt.Println("download finished")
			fmt.Println("————————————————————————————————————————————————————")
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
		utils.DebugLog("sp.TotalSize", sp.TotalSize)
		if info.Size() == sp.TotalSize {
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
			if sp.DownloadSize >= sp.TotalSize {
				fName := fInfo.FileName
				if fName == "" {
					fName = fileHash
				}
				filePath := file.GetDownloadTmpPath(fileHash, fName, fInfo.SavePath)
				if CheckFileOver(fileHash, filePath) {
					DoneDownload(fileHash, fName, fInfo.SavePath)
					DownloadFileMap.Delete(fileHash)
					DownloadSpeedOfProgress.Delete(fileHash)
					return true, 1.0
				}
				reCount = 5
				time.Sleep(time.Second * 2)
				checkAgain(fileHash)
				return true, 1
			}
			return false, float32(sp.DownloadSize) / float32(sp.TotalSize)
		}
		return false, 0
	}
	utils.ErrorLog("download error, failed to find the task, request download again")
	return false, 0

}

// DownloadProgress
func DownloadProgress(fielHash string, size uint64) {
	if s, ok := DownloadSpeedOfProgress.Load(fielHash); ok {
		sp := s.(*DownloadSP)
		sp.DownloadSize += int64(size)
		p := float32(sp.DownloadSize) / float32(sp.TotalSize) * 100
		fmt.Printf("downloaded：%.2f %% \n", p)
		setting.DownProssMap.Store(fielHash, p)
		setting.ShowProgress(p)
		if sp.DownloadSize >= sp.TotalSize {
			go CheckDownloadOver(fielHash)
		}
	}
}
