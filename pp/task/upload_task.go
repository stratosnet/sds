package task

import (
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/utils"
	"sync"
)


var urwmutex sync.RWMutex

// UploadSliceTask
type UploadSliceTask struct {
	TaskID          string
	FileHash        string
	SliceNumAddr    *protos.SliceNumAddr    // upload PP address and sliceNumber
	SliceOffsetInfo *protos.SliceOffsetInfo
	FileCRC         uint32
	Data            []byte
	SliceTotalSize  uint64
}

// MAXSLICE max slice number that can upload concurrently for a single file
const MAXSLICE = 50

// UpFileIng uploadingfile
type UpFileIng struct {
	UPING    int
	Slices   []*protos.SliceNumAddr
	TaskID   string
	FileHash string
	UpChan   chan bool
}

// UpIngMap UpIng
var UpIngMap = &sync.Map{}

// UpProgress
type UpProgress struct {
	Total     int64
	HasUpload int64
}

// UpLoadProgressMap
var UpLoadProgressMap = &sync.Map{}

// GetUploadSliceTask
func GetUploadSliceTask(pp *protos.SliceNumAddr, fileHash, taskID string) *UploadSliceTask {
	filePath := file.GetFilePath(fileHash)
	utils.DebugLog("offsetStart =", pp.SliceOffset.SliceOffsetStart, "offsetEnd", pp.SliceOffset.SliceOffsetEnd)
	utils.DebugLog("sliceNumber", pp.SliceNumber)
	startOffsize := pp.SliceOffset.SliceOffsetStart
	endOffsize := pp.SliceOffset.SliceOffsetEnd
	if file.GetFileInfo(filePath) == nil {
		utils.ErrorLog("wrong file path")
		return nil
	}
	if uint64(file.GetFileInfo(filePath).Size()) < endOffsize {
		endOffsize = uint64(file.GetFileInfo(filePath).Size())
	}

	offset := &protos.SliceOffset{
		SliceOffsetStart: startOffsize,
		SliceOffsetEnd:   endOffsize,
	}
	data := file.GetFileData(filePath, offset)
	sl := &protos.SliceOffsetInfo{
		SliceHash:   utils.CalcHash(data),
		SliceOffset: offset,
	}
	tk := &UploadSliceTask{
		TaskID:          taskID,
		FileHash:        fileHash,
		SliceNumAddr:    pp,
		SliceOffsetInfo: sl,
		FileCRC:         utils.CalcFileCRC32(filePath),
		Data:            file.GetFileData(filePath, offset),
		SliceTotalSize:  pp.SliceOffset.SliceOffsetEnd - pp.SliceOffset.SliceOffsetStart,
	}
	return tk
}

// SaveUploadFile
func SaveUploadFile(target *protos.ReqUploadFileSlice) bool {
	return file.SaveSliceData(target.Data, target.SliceInfo.SliceHash, target.SliceInfo.SliceOffset.SliceOffsetStart)
}
