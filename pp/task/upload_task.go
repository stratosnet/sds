package task

import (
	"encoding/json"
	"math/rand"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/encryption"
	"github.com/stratosnet/sds/utils/encryption/hdkey"
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
	SpP2pAddress    string
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
	FileCRC  uint32
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

func CleanUpConnMap(fileHash string) {
	client.UpConnMap.Range(func(k, v interface{}) bool {
		if strings.HasPrefix(k.(string), fileHash) {
			client.UpConnMap.Delete(k.(string))
		}
		return true
	})
}

// GetUploadSliceTask
func GetUploadSliceTask(pp *protos.SliceNumAddr, fileHash, taskID, spP2pAddress string, isVideoStream, isEncrypted bool, fileCRC uint32) *UploadSliceTask {
	if isVideoStream {
		return GetUploadSliceTaskStream(pp, fileHash, taskID, spP2pAddress, fileCRC)
	} else {
		return GetUploadSliceTaskFile(pp, fileHash, taskID, spP2pAddress, isEncrypted, fileCRC)
	}
}

func GetUploadSliceTaskFile(pp *protos.SliceNumAddr, fileHash, taskID, spP2pAddress string, isEncrypted bool, fileCRC uint32) *UploadSliceTask {

	utils.DebugLogf("sliceNumber %v  offsetStart = %v  offsetEnd = %v", pp.SliceNumber, pp.SliceOffset.SliceOffsetStart, pp.SliceOffset.SliceOffsetEnd)
	startOffset := pp.SliceOffset.SliceOffsetStart
	endOffset := pp.SliceOffset.SliceOffsetEnd

	var fileSize uint64
	var filePath string

	remote := file.IsFileRpcRemote(fileHash)
	if !remote {
		// in case of local file
		filePath = file.GetFilePath(fileHash)
		fileInfo := file.GetFileInfo(filePath)
		if fileInfo == nil {
			utils.ErrorLog("wrong file path")
			return nil
		}
		fileSize = uint64(fileInfo.Size())
	} else {
		// in case of remote (rpc) file
		fileSize = file.GetRemoteFileSize(fileHash)
	}

	if fileSize < endOffset {
		endOffset = fileSize
	}
	offset := &protos.SliceOffset{
		SliceOffsetStart: startOffset,
		SliceOffsetEnd:   endOffset,
	}

	var rawData []byte
	if !remote {
		rawData = file.GetFileData(filePath, offset)
	} else {
		rawData = file.GetRemoteFileData(fileHash, offset)
	}

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
	sliceHash := utils.CalcSliceHash(data, fileHash, pp.SliceNumber)

	sl := &protos.SliceOffsetInfo{
		SliceHash: sliceHash,
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
		FileCRC:         fileCRC,
		Data:            data,
		SliceTotalSize:  dataSize,
		SpP2pAddress:    spP2pAddress,
	}
	file.SaveTmpSliceData(fileHash, sliceHash, data)
	return tk
}

func GetUploadSliceTaskStream(pp *protos.SliceNumAddr, fileHash, taskID, spP2pAddress string, fileCRC uint32) *UploadSliceTask {
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

	sliceHash := utils.CalcSliceHash(data, fileHash, pp.SliceNumber)
	offset := &protos.SliceOffset{
		SliceOffsetStart: uint64(0),
		SliceOffsetEnd:   sliceTotalSize,
	}
	sl := &protos.SliceOffsetInfo{
		SliceHash:   sliceHash,
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
		FileCRC:         fileCRC,
		Data:            data,
		SliceTotalSize:  sliceTotalSize,
		SpP2pAddress:    spP2pAddress,
	}
	file.SaveTmpSliceData(fileHash, sliceHash, data)
	return tk
}

func GetReuploadSliceTask(pp *protos.SliceHashAddr, fileHash, taskID, spP2pAddress string) *UploadSliceTask {
	utils.DebugLogf("  fileHash %s sliceNumber %v, sliceHash %s",
		fileHash, pp.SliceNumber, pp.SliceHash)

	rawData := file.GetSliceDataFromTmp(fileHash, pp.SliceHash)

	if rawData == nil {
		utils.ErrorLogf("Failed to find the file slice in temp folder for fileHash %s sliceNumber %v, sliceHash %s",
			fileHash, pp.SliceNumber, pp.SliceHash)
		return nil
	}

	data := rawData
	dataSize := uint64(len(data))

	sl := &protos.SliceOffsetInfo{
		SliceHash: pp.SliceHash,
		SliceOffset: &protos.SliceOffset{
			SliceOffsetStart: 0,
			SliceOffsetEnd:   dataSize,
		},
	}

	tk := &UploadSliceTask{
		TaskID:   taskID,
		FileHash: fileHash,
		SliceNumAddr: &protos.SliceNumAddr{
			SliceNumber: pp.SliceNumber,
			SliceOffset: pp.SliceOffset,
			PpInfo:      pp.PpInfo,
		},
		SliceOffsetInfo: sl,
		Data:            data,
		SliceTotalSize:  dataSize,
		SpP2pAddress:    spP2pAddress,
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
