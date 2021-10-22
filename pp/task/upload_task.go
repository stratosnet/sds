package task

import (
	"encoding/json"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/encryption"
	"github.com/stratosnet/sds/utils/encryption/hdkey"
	"math/rand"
	"sync"
)

var urwmutex sync.RWMutex

// UploadSliceTask
type UploadSliceTask struct {
	TaskID          string
	FileHash        string
	SliceNumAddr    *protos.SliceNumAddr // upload PP address and sliceNumber
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

// UploadProgressMap
var UploadProgressMap = &sync.Map{}

// GetUploadSliceTask
func GetUploadSliceTask(pp *protos.SliceNumAddr, fileHash, taskID string, isVideoStream, isEncrypted bool) *UploadSliceTask {
	if isVideoStream {
		return GetUploadSliceTaskStream(pp, fileHash, taskID)
	} else {
		return GetUploadSliceTaskFile(pp, fileHash, taskID, isEncrypted)
	}
}

func GetUploadSliceTaskFile(pp *protos.SliceNumAddr, fileHash, taskID string, isEncrypted bool) *UploadSliceTask {
	filePath := file.GetFilePath(fileHash)
	utils.DebugLog("offsetStart =", pp.SliceOffset.SliceOffsetStart, "offsetEnd", pp.SliceOffset.SliceOffsetEnd)
	utils.DebugLog("sliceNumber", pp.SliceNumber)
	startOffset := pp.SliceOffset.SliceOffsetStart
	endOffset := pp.SliceOffset.SliceOffsetEnd
	if file.GetFileInfo(filePath) == nil {
		utils.ErrorLog("wrong file path")
		return nil
	}

	if uint64(file.GetFileInfo(filePath).Size()) < endOffset {
		endOffset = uint64(file.GetFileInfo(filePath).Size())
	}

	offset := &protos.SliceOffset{
		SliceOffsetStart: startOffset,
		SliceOffsetEnd:   endOffset,
	}
	rawData := file.GetFileData(filePath, offset)

	// Encrypt slice data if required
	data := rawData
	if isEncrypted {
		var err error
		data, err = encryptSliceData(rawData)
		if err != nil {
			utils.ErrorLog("Couldn't encrypt slice data", err)
			return nil
		}
	}
	dataSize := uint64(len(data))

	sl := &protos.SliceOffsetInfo{
		SliceHash: utils.CalcSliceHash(data, fileHash),
		SliceOffset: &protos.SliceOffset{
			SliceOffsetStart: 0,
			SliceOffsetEnd:   dataSize,
		},
	}

	tk := &UploadSliceTask{
		TaskID:          taskID,
		FileHash:        fileHash,
		SliceNumAddr:    pp,
		SliceOffsetInfo: sl,
		FileCRC:         utils.CalcFileCRC32(filePath),
		Data:            data,
		SliceTotalSize:  dataSize,
	}
	return tk
}

func GetUploadSliceTaskStream(pp *protos.SliceNumAddr, fileHash, taskID string) *UploadSliceTask {
	videoFolder := file.GetVideoTmpFolder(fileHash)
	videoSliceInfo := file.HlsInfoMap[fileHash]
	var data []byte
	var sliceTotalSize uint64

	if pp.SliceNumber == 1 {
		jsonStr, _ := json.Marshal(videoSliceInfo)
		data = jsonStr
		sliceTotalSize = uint64(len(data))
	} else if pp.SliceNumber < videoSliceInfo.StartSliceNumber {
		data = file.GetDumpySliceData(fileHash, pp.SliceNumber)
		sliceTotalSize = uint64(len(data))
	} else {
		var sliceName string
		sliceName = videoSliceInfo.SliceToSegment[pp.SliceNumber]
		slicePath := videoFolder + "/" + sliceName
		if file.GetFileInfo(slicePath) == nil {
			utils.ErrorLog("wrong file path")
			return nil
		}
		data = file.GetWholeFileData(slicePath)
		sliceTotalSize = uint64(file.GetFileInfo(slicePath).Size())
	}

	utils.DebugLog("sliceNumber", pp.SliceNumber)

	offset := &protos.SliceOffset{
		SliceOffsetStart: uint64(0),
		SliceOffsetEnd:   sliceTotalSize,
	}
	sl := &protos.SliceOffsetInfo{
		SliceHash:   utils.CalcSliceHash(data, fileHash),
		SliceOffset: offset,
	}
	SliceNumAddr := &protos.SliceNumAddr{
		SliceNumber: pp.SliceNumber,
		SliceOffset: offset,
		PpInfo:      pp.PpInfo,
	}
	pp.SliceOffset = offset
	tk := &UploadSliceTask{
		TaskID:          taskID,
		FileHash:        fileHash,
		SliceNumAddr:    SliceNumAddr,
		SliceOffsetInfo: sl,
		FileCRC:         utils.CalcFileCRC32(file.GetFilePath(fileHash)),
		Data:            data,
		SliceTotalSize:  sliceTotalSize,
	}
	return tk
}

// SaveUploadFile
func SaveUploadFile(target *protos.ReqUploadFileSlice) bool {
	return file.SaveSliceData(target.Data, target.SliceInfo.SliceHash, target.SliceInfo.SliceOffset.SliceOffsetStart)
}

func encryptSliceData(rawData []byte) ([]byte, error) {
	hdKeyNonce := rand.Uint32()
	if hdKeyNonce > hdkey.HardenedKeyStart {
		hdKeyNonce -= hdkey.HardenedKeyStart
	}
	aesNonce := rand.Uint64()

	key, err := hdkey.MasterKeyForSliceEncryption(setting.WalletPrivateKey, hdKeyNonce)
	if err != nil {
		return nil, err
	}

	encryptedData, err := encryption.EncryptAES(key.PrivateKey(), rawData, aesNonce)
	if err != nil {
		return nil, err
	}

	encryptedSlice := &protos.EncryptedSlice{
		HdkeyNonce: hdKeyNonce,
		AesNonce:   aesNonce,
		Data:       encryptedData,
		RawSize:    uint64(len(rawData)),
	}
	return proto.Marshal(encryptedSlice)
}
