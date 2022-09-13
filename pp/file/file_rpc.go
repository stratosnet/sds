package file

import (
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/api/rpc"
	"io"
	"strings"
	"sync"
	"time"
)

const WAIT_TIMEOUT time.Duration = 5

var (
	reFileMutex sync.Mutex

	downDataMutex sync.Mutex

	// key(fileHash + fileReqId) : value(fileSize)
	rpcFileInfoMap = &sync.Map{}

	// key(fileHash + fileReqId) : value(*rpc.Result)
	rpcFileEvent = &sync.Map{}

	// key(fileHash) : value(pipe)
	rpcUploadPipes = make(map[string]pipe)

	// key(fileHash + file) : value(downloadReady)
	rpcDownloadReady = &sync.Map{}

	// key(fileHash) : value(download file data)
	rpcDownloadData = &sync.Map{}

	// gracefully close download session
	rpcDownSessionClosing = &sync.Map{}

	// key(walletAddr + reqid): value(file list result)
	rpcFileListResult = &sync.Map{}

	rpcFileShareResult = &sync.Map{}

	// key(wallet + reqid) : value(*rpc.GetOzoneResult)
	rpcOzone = &sync.Map{}
)

type pipe struct {
	reader *io.PipeReader
	writer *io.PipeWriter
}

// IsFileRpcRemote
func IsFileRpcRemote(key string) bool {
	str := fileMap[key]
	if str == "" {
		return false
	}
	return strings.Split(str, ":")[0] == "rpc"
}

// GetRemoteFileData
func GetRemoteFileData(hash string, offset *protos.SliceOffset) []byte {
	// input check
	if offset == nil {
		return nil
	}

	// compose event, as well notify the remote user
	r := &rpc.Result{
		Return:      rpc.UPLOAD_DATA,
		OffsetStart: &offset.SliceOffsetStart,
		OffsetEnd:   &offset.SliceOffsetEnd,
	}

	// send event and open the pipe for coming data

	reFileMutex.Lock()
	rpcFileEvent.Store(hash, r)
	var p pipe
	p.reader, p.writer = io.Pipe()
	rpcUploadPipes[hash] = p
	reFileMutex.Unlock()

	// read on the pipe
	data := make([]byte, offset.SliceOffsetEnd-offset.SliceOffsetStart)
	var cursor []byte
	var read uint64
	var done = make(chan bool)

	cursor = data[:]

	go func() {
		for {
			n, err := p.reader.Read(cursor)
			if err != nil {
				done <- false
				return
			}
			read = read + uint64(n)
			cursor = data[read:]
			if read >= offset.SliceOffsetEnd-offset.SliceOffsetStart {
				done <- true
				return
			}
		}
	}()

	select {
	case <-time.After(WAIT_TIMEOUT * time.Second):
		return nil
	case s := <-done:
		if s {
			return []byte(data)
		} else {
			return nil
		}
	}
}

// GetDownloadFileData
func GetDownloadFileData(key string) []byte {
	var data []byte

	for {
		downDataMutex.Lock()
		d, found := rpcDownloadData.LoadAndDelete(key)
		downDataMutex.Unlock()
		if found && d != nil {
			data = d.([]byte)
			break
		}
	}

	return data
}

// SaveRemoteFileData
func SaveRemoteFileData(key string, data []byte, offset uint64) bool {
	if data == nil {
		return false
	}

	// in case the COMM is broken, gracefully close the download session
	closing, found := rpcDownSessionClosing.LoadAndDelete(key)
	if found && closing.(bool) {
		return false
	}

	wmutex.Lock()
	defer wmutex.Unlock()
	// 1. send the event rpc.DOWNLOAD_OK
	offsetend := offset + uint64(len(data))
	result := rpc.Result{
		Return:      rpc.DOWNLOAD_OK,
		OffsetStart: &offset,
		OffsetEnd:   &offsetend,
	}

	downDataMutex.Lock()
	SetRemoteFileResult(key, result)

	// 2. download file data -> map
	rpcDownloadData.Store(key, data)
	downDataMutex.Unlock()

	// 3. need to wait the reply from rpc comm confirmed
	return WaitDownloadSliceDone(key)
}

// GetRemoteFileSize
func GetRemoteFileSize(hash string) uint64 {
	if f, ok := rpcFileInfoMap.Load(hash); ok {
		return f.(uint64)
	}
	return 0
}

// SendFileDataBack the rpc handler writes data to slice upload task
func SendFileDataBack(hash string, content []byte) {
	reFileMutex.Lock()

	if w, found := rpcUploadPipes[hash]; found && w.writer != nil {
		rpcUploadPipes[hash].writer.Write(content)
	}

	reFileMutex.Unlock()
}

// SetRemoteFileResult a result is given to the remote client
func SetRemoteFileResult(key string, result rpc.Result) {
	rpcFileEvent.Store(key, &result)
}

// SaveRemoteFileHash
func SaveRemoteFileHash(hash, fileName string, fileSize uint64) {
	fileMap[hash] = "rpc:" + fileName
	rpcFileInfoMap.Store(hash, fileSize)
}

// SetRemoteFileInfo
func SetRemoteFileInfo(hash string, size uint64) {
	rpcFileInfoMap.Store(hash, size)
}

// GetRemoteFileEvent
func GetRemoteFileEvent(key string) (*rpc.Result, bool) {
	result, loaded := rpcFileEvent.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.Result), loaded
	} else {
		return nil, loaded
	}
}

// SetDownloadSliceDone a result is given to the remote client
func SetDownloadSliceDone(key string) {
	rpcDownloadReady.Store(key, true)
}

// WaitDownloadSliceDone
func WaitDownloadSliceDone(key string) bool {

	rpcDownloadReady.Store(key, false)

	var done = make(chan bool)
	go func() {

		for {
			value, found := rpcDownloadReady.Load(key)
			if found && value.(bool) {
				done <- true
				return
			}
		}
	}()

	select {
	case <-time.After(WAIT_TIMEOUT * time.Second):
		return false
	case <-done:
		return true
	}
}

// GetRemoteFileInfo
func GetRemoteFileInfo(hash string) uint64 {
	SetRemoteFileResult(hash, rpc.Result{Return: rpc.DL_OK_ASK_INFO})
	var fileSize uint64
	for {
		fileSize = GetRemoteFileSize(hash)
		if fileSize != 0 {
			break
		}
	}
	return fileSize
}

// CleanFileHash
func CleanFileHash(key string) {
	reFileMutex.Lock()
	rpcFileInfoMap.Delete(key)
	rpcFileEvent.Delete(key)
	rpcDownloadReady.Delete(key)
	rpcDownloadData.Delete(key)
	ClearFileMap(key)
	reFileMutex.Unlock()
}

// CloseDownloadSession
func CloseDownloadSession(key string) {
	rpcDownSessionClosing.Store(key, true)
}

// GetFileListResult
func GetFileListResult(key string) (*rpc.FileListResult, bool) {
	result, loaded := rpcFileListResult.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.FileListResult), loaded
	}
	return nil, loaded
}

// SetFileListResult
func SetFileListResult(key string, result *rpc.FileListResult) {
	if result != nil {
		rpcFileListResult.Store(key, result)
	}
}

// GetFileShareResult
func GetFileShareResult(key string) (*rpc.FileShareResult, bool) {
	result, loaded := rpcFileShareResult.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.FileShareResult), loaded
	}
	return nil, loaded
}

// SetFileShareResult
func SetFileShareResult(key string, result *rpc.FileShareResult) {
	if result != nil {
		rpcFileShareResult.Store(key, result)
	}
}

// GetQueryOzoneResult
func GetQueryOzoneResult(key string) (*rpc.GetOzoneResult, bool) {
	result, loaded := rpcOzone.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.GetOzoneResult), loaded
	}
	return nil, loaded
}

// SetQueryOzoneResult
func SetQueryOzoneResult(key string, result *rpc.GetOzoneResult) {
	if result != nil {
		rpcOzone.Store(key, result)
	}
}
