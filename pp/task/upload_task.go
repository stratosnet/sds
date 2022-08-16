package task

import (
	"context"
	"encoding/json"
	"math/rand"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
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
func GetUploadSliceTask(ctx context.Context, ppNode *protos.SliceNumAddr, fileHash, taskID, spP2pAddress string, isVideoStream, isEncrypted bool, fileCRC uint32) *UploadSliceTask {
	if isVideoStream {
		return GetUploadSliceTaskStream(ctx, ppNode, fileHash, taskID, spP2pAddress, fileCRC)
	} else {
		return GetUploadSliceTaskFile(ctx, ppNode, fileHash, taskID, spP2pAddress, isEncrypted, fileCRC)
	}
}

func GetUploadSliceTaskFile(ctx context.Context, ppNode *protos.SliceNumAddr, fileHash, taskID, spP2pAddress string, isEncrypted bool, fileCRC uint32) *UploadSliceTask {

	pp.DebugLogf(ctx, "sliceNumber %v  offsetStart = %v  offsetEnd = %v", ppNode.SliceNumber, ppNode.SliceOffset.SliceOffsetStart, ppNode.SliceOffset.SliceOffsetEnd)
	startOffset := ppNode.SliceOffset.SliceOffsetStart
	endOffset := ppNode.SliceOffset.SliceOffsetEnd

	var fileSize uint64
	var filePath string

	remote := file.IsFileRpcRemote(fileHash)
	if !remote {
		// in case of local file
		filePath = file.GetFilePath(fileHash)
		fileInfo := file.GetFileInfo(filePath)
		if fileInfo == nil {
			pp.ErrorLog(ctx, "wrong file path")
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
			pp.ErrorLog(ctx, "Couldn't encrypt slice data", err)
			return nil
		}
	}
	dataSize := uint64(len(data))
	sliceHash := utils.CalcSliceHash(data, fileHash, ppNode.SliceNumber)

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
		SliceNumAddr:    ppNode,
		SliceOffsetInfo: sl,
		FileCRC:         fileCRC,
		Data:            data,
		SliceTotalSize:  dataSize,
		SpP2pAddress:    spP2pAddress,
	}
	file.SaveTmpSliceData(ctx, fileHash, sliceHash, data)
	return tk
}

func GetUploadSliceTaskStream(ctx context.Context, ppNode *protos.SliceNumAddr, fileHash, taskID, spP2pAddress string, fileCRC uint32) *UploadSliceTask {
	videoFolder := file.GetVideoTmpFolder(fileHash)
	videoSliceInfo := file.HlsInfoMap[fileHash]
	var data []byte
	var sliceTotalSize uint64

	if ppNode.SliceNumber == 1 {
		jsonStr, _ := json.Marshal(videoSliceInfo)
		data = jsonStr
		sliceTotalSize = uint64(len(data))
	} else if ppNode.SliceNumber < videoSliceInfo.StartSliceNumber {
		data = file.GetDumpySliceData(fileHash, ppNode.SliceNumber)
		sliceTotalSize = uint64(len(data))
	} else {
		var sliceName string
		sliceName = videoSliceInfo.SliceToSegment[ppNode.SliceNumber]
		slicePath := videoFolder + "/" + sliceName
		if file.GetFileInfo(slicePath) == nil {
			pp.ErrorLog(ctx, "wrong file path")
			return nil
		}
		data = file.GetWholeFileData(slicePath)
		sliceTotalSize = uint64(file.GetFileInfo(slicePath).Size())
	}

	pp.DebugLog(ctx, "sliceNumber", ppNode.SliceNumber)

	sliceHash := utils.CalcSliceHash(data, fileHash, ppNode.SliceNumber)
	offset := &protos.SliceOffset{
		SliceOffsetStart: uint64(0),
		SliceOffsetEnd:   sliceTotalSize,
	}
	sl := &protos.SliceOffsetInfo{
		SliceHash:   sliceHash,
		SliceOffset: offset,
	}
	SliceNumAddr := &protos.SliceNumAddr{
		SliceNumber: ppNode.SliceNumber,
		SliceOffset: offset,
		PpInfo:      ppNode.PpInfo,
	}
	ppNode.SliceOffset = offset
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
	file.SaveTmpSliceData(ctx, fileHash, sliceHash, data)
	return tk
}

func GetReuploadSliceTask(ppNode *protos.SliceHashAddr, fileHash, taskID, spP2pAddress string) *UploadSliceTask {
	utils.DebugLogf("  fileHash %s sliceNumber %v, sliceHash %s",
		fileHash, ppNode.SliceNumber, ppNode.SliceHash)

	rawData := file.GetSliceDataFromTmp(fileHash, ppNode.SliceHash)

	if rawData == nil {
		utils.ErrorLogf("Failed to find the file slice in temp folder for fileHash %s sliceNumber %v, sliceHash %s",
			fileHash, ppNode.SliceNumber, ppNode.SliceHash)
		return nil
	}

	data := rawData
	dataSize := uint64(len(data))

	sl := &protos.SliceOffsetInfo{
		SliceHash: ppNode.SliceHash,
		SliceOffset: &protos.SliceOffset{
			SliceOffsetStart: 0,
			SliceOffsetEnd:   dataSize,
		},
	}

	tk := &UploadSliceTask{
		TaskID:   taskID,
		FileHash: fileHash,
		SliceNumAddr: &protos.SliceNumAddr{
			SliceNumber: ppNode.SliceNumber,
			SliceOffset: ppNode.SliceOffset,
			PpInfo:      ppNode.PpInfo,
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
