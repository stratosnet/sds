package file

import (
	"context"
	b64 "encoding/base64"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/metrics"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/api/rpc"
)

const NUMBER_OF_UPLOAD_CHAN_BUFFER = 5

var (
	reFileMutex sync.Mutex

	upSliceMutex sync.Mutex

	eventMutex sync.Mutex

	// key(fileHash + fileReqId) : value(fileSize)
	rpcFileInfoMap = &sync.Map{}

	// key(fileHash + fileReqId) : value(chan *rpc.Result)
	rpcFileEventChan = &sync.Map{}

	// key(fileHash) : value(chan []byte)
	rpcUploadDataChan = &sync.Map{}

	// key(fileHash + file) : value(downloadReady)
	rpcDownloadReady = &sync.Map{}

	// key(fileHash + file) : value(chan uint64)
	rpcDownloadFileInfo = &sync.Map{}

	// gracefully close download session
	rpcDownSessionClosing = &sync.Map{}

	// key(walletAddr + reqid): value(file list result)
	rpcFileListResult = &sync.Map{}

	rpcFileShareResult = &sync.Map{}

	// key(wallet + reqid) : value(*rpc.GetOzoneResult)
	rpcOzone = &sync.Map{}

	// wait for the next request from client per message
	RpcWaitTimeout time.Duration
)

func IsFileRpcRemote(key string) bool {
	str := fileMap[key]
	if str == "" {
		return false
	}
	return strings.Split(str, ":")[0] == "rpc"
}

// SubscribeGetRemoteFileData application subscribes to remote file data and waits for remote user's feedback
func SubscribeGetRemoteFileData(key string) chan []byte {
	event, found := rpcUploadDataChan.Load(key)
	if !found {
		event = make(chan []byte, NUMBER_OF_UPLOAD_CHAN_BUFFER)
		rpcUploadDataChan.Store(key, event)
	}
	return event.(chan []byte)
}

// UnsubscribeGetRemoteFileData unsubscribe after the application finishes receiving the slice of file data
func UnsubscribeGetRemoteFileData(key string) {
	rpcUploadDataChan.Delete(key)
}

// GetRemoteFileData application calls this func to fetch a (sub)slice of file data from remote user
func GetRemoteFileData(hash string, offset *protos.SliceOffset) []byte {
	upSliceMutex.Lock()
	defer upSliceMutex.Unlock()

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
	SetRemoteFileResult(hash, *r)

	// read on the pipe
	data := make([]byte, offset.SliceOffsetEnd-offset.SliceOffsetStart)
	var cursor []byte
	var read uint64 = 0

	cursor = data[:]
OuterFor:
	for {
		parentCtx := context.Background()
		ctx, cancel := context.WithTimeout(parentCtx, RpcWaitTimeout)

		select {
		case <-ctx.Done():
			cancel()
			return nil
		case subSlice := <-SubscribeGetRemoteFileData(hash):
			metrics.UploadPerformanceLogNow(hash + ":RCV_SUBSLICE_RPC:" + strconv.FormatInt(int64(offset.SliceOffsetStart), 10))
			copy(cursor, subSlice)
			// one piece to be sent to client
			read = read + uint64(len(subSlice))
			cursor = data[read:]
			if read >= offset.SliceOffsetEnd-offset.SliceOffsetStart {
				UnsubscribeGetRemoteFileData(hash)
				cancel()
				break OuterFor
			}
		}
	}

	return data
}

func CacheRemoteFileData(fileHash string, offset *protos.SliceOffset, fileName string) error {
	upSliceMutex.Lock()
	defer upSliceMutex.Unlock()

	// compose event, as well notify the remote user
	r := &rpc.Result{
		Return:      rpc.UPLOAD_DATA,
		OffsetStart: &offset.SliceOffsetStart,
		OffsetEnd:   &offset.SliceOffsetEnd,
	}

	// send event and open the pipe for coming data
	SetRemoteFileResult(fileHash, *r)

	fileMg, err := OpenTmpFile(fileHash, fileName)
	if err != nil {
		return errors.Wrap(err, "failed opening temp file")
	}
	defer func() {
		_ = fileMg.Close()
	}()

	var read int64 = 0

OuterFor:
	for {
		parentCtx := context.Background()
		ctx, cancel := context.WithTimeout(parentCtx, RpcWaitTimeout)

		select {
		case <-ctx.Done():
			cancel()
			return errors.New("timeout waiting uploaded sub-slice")
		case subSlice := <-SubscribeGetRemoteFileData(fileHash):
			metrics.UploadPerformanceLogNow(fileHash + ":RCV_SUBSLICE_RPC:" + strconv.FormatInt(int64(offset.SliceOffsetStart), 10))
			err = WriteFile(subSlice, read, fileMg)
			if err != nil {
				cancel()
				return errors.Wrap(err, "failed writing file")
			}
			read = read + int64(len(subSlice))
			if read >= int64(offset.SliceOffsetEnd-offset.SliceOffsetStart) {
				UnsubscribeGetRemoteFileData(fileHash)
				cancel()
				break OuterFor
			}
		}
	}

	return nil
}

// SendFileDataBack rpc server feeds file data from remote user to application
func SendFileDataBack(hash string, content []byte) {
	ch, found := rpcUploadDataChan.Load(hash)
	if found {
		select {
		case ch.(chan []byte) <- content:
		default:
			UnsubscribeGetRemoteFileData(hash)
		}
	}
}

// SaveRemoteFileData application calls this func to send a slice of file data to remote user during download process
func SaveRemoteFileData(key, fileName string, data []byte, offset uint64) error {
	if data == nil {
		return errors.New("invalid input data")
	}

	// in case the COMM is broken, gracefully close the download session
	closing, found := rpcDownSessionClosing.LoadAndDelete(key)
	if found && closing.(bool) {
		return errors.New("closing session")
	}

	wmutex.Lock()
	defer wmutex.Unlock()
	// 1. send the event rpc.DOWNLOAD_OK
	offsetend := offset + uint64(len(data))
	result := rpc.Result{
		Return:      rpc.DOWNLOAD_OK,
		OffsetStart: &offset,
		OffsetEnd:   &offsetend,
		FileData:    b64.StdEncoding.EncodeToString(data),
		FileName:    fileName,
	}

	SetRemoteFileResult(key, result)
	return WaitDownloadSliceDone(key)
}

func GetRemoteFileSize(hash string) uint64 {
	if f, ok := rpcFileInfoMap.Load(hash); ok {
		return f.(uint64)
	}
	return 0
}

func SaveRemoteFileHash(hash, fileName string, fileSize uint64) {
	reFileMutex.Lock()
	defer reFileMutex.Unlock()

	fileMap[hash] = "rpc:" + fileName
	rpcFileInfoMap.Store(hash, fileSize)
}

// SubscribeRemoteFileEvent rpc server subscribes to events from application. Now, result of operation is the only event
func SubscribeRemoteFileEvent(key string) chan *rpc.Result {
	event := make(chan *rpc.Result)
	eventMutex.Lock()
	defer eventMutex.Unlock()
	rpcFileEventChan.Store(key, event)
	return event
}

// UnsubscribeRemoteFileEvent rpc server unsubscribes to event from application. Now, result of operation is the only event
func UnsubscribeRemoteFileEvent(key string) {
	eventMutex.Lock()
	defer eventMutex.Unlock()
	rpcFileEventChan.Delete(key)
}

// SetRemoteFileResult application sends the result of previous operation to rpc server
func SetRemoteFileResult(key string, result rpc.Result) {
	eventMutex.Lock()
	defer eventMutex.Unlock()
	ch, found := rpcFileEventChan.Load(key)
	if found {
		select {
		case ch.(chan *rpc.Result) <- &result:
		default:
			rpcFileEventChan.Delete(key)
		}
	}
}

func SubscribeDownloadSliceDone(key string) chan bool {
	done := make(chan bool)
	rpcDownloadReady.Store(key, done)
	return done
}

func UnsubscribeDownloadSliceDone(key string) {
	rpcDownloadReady.Delete(key)
}

// SetDownloadSliceDone rpc server tells the remote user has received last downloaded slice
func SetDownloadSliceDone(key string) {
	ch, found := rpcDownloadReady.Load(key)
	if found {
		select {
		case ch.(chan bool) <- true:
		default:
			UnsubscribeDownloadSliceDone(key)
		}
	}
}

func SubscribeGetSignature(key string) chan []byte {
	sig := make(chan []byte)
	rpcDownloadReady.Store(key, sig)
	return sig
}

func UnsubscribeGetSignature(key string) {
	rpcDownloadReady.Delete(key)
}

func GetSignatureFromRemote(key string) []byte {
	parentCtx := context.Background()
	ctx, cancel := context.WithTimeout(parentCtx, RpcWaitTimeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return nil
	case signature := <-SubscribeGetSignature(key):
		UnsubscribeDownloadSliceDone(key)
		return signature
	}
}

func SetSignature(key string, sig []byte) {
	ch, found := rpcDownloadReady.Load(key)
	if found {
		select {
		case ch.(chan []byte) <- sig:
		default:
			return
		}
	}
}

// WaitDownloadSliceDone application waits for remote user to tell that the downloaded slice received
func WaitDownloadSliceDone(key string) error {
	var done bool
	parentCtx := context.Background()
	ctx, cancel := context.WithTimeout(parentCtx, RpcWaitTimeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return errors.New("timeout waiting download slice")
	case done = <-SubscribeDownloadSliceDone(key):
		UnsubscribeDownloadSliceDone(key)
		if done {
			return nil
		}
		return errors.New("download slice invalid state")
	}
}

func SubscribeDownloadFileInfo(key string) chan uint64 {
	fileSize := make(chan uint64)
	rpcDownloadFileInfo.Store(key, fileSize)
	return fileSize
}

func UnsubscribeDownloadFileInfo(key string) {
	rpcDownloadFileInfo.Delete(key)
}

func GetRemoteFileInfo(key, reqId string) uint64 {
	SetRemoteFileResult(key, rpc.Result{ReqId: reqId, Return: rpc.DL_OK_ASK_INFO})
	var fileSize uint64
	parentCtx := context.Background()
	ctx, cancel := context.WithTimeout(parentCtx, RpcWaitTimeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return 0
	case fileSize = <-SubscribeDownloadFileInfo(key):
		UnsubscribeDownloadFileInfo(key)
	}
	return fileSize
}

func SetRemoteFileInfo(key string, size uint64) {
	reFileMutex.Lock()
	defer reFileMutex.Unlock()
	ch, found := rpcDownloadFileInfo.Load(key)
	if found {
		select {
		case ch.(chan uint64) <- size:
		default:
			rpcDownloadFileInfo.Delete(key)
		}
	}
}

func CleanFileHash(key string) {
	reFileMutex.Lock()
	defer reFileMutex.Unlock()
	rpcFileInfoMap.Delete(key)
	ClearFileMap(key)
}

func CloseDownloadSession(key string) {
	rpcDownSessionClosing.Store(key, true)
}

func GetFileListResult(key string) (*rpc.FileListResult, bool) {
	result, loaded := rpcFileListResult.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.FileListResult), loaded
	}
	return nil, loaded
}

func SetFileListResult(key string, result *rpc.FileListResult) {
	if result != nil {
		rpcFileListResult.Store(key, result)
	}
}

func GetFileShareResult(key string) (*rpc.FileShareResult, bool) {
	result, loaded := rpcFileShareResult.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.FileShareResult), loaded
	}
	return nil, loaded
}

func SetFileShareResult(key string, result *rpc.FileShareResult) {
	if result != nil {
		rpcFileShareResult.Store(key, result)
	}
}

func GetQueryOzoneResult(key string) (*rpc.GetOzoneResult, bool) {
	result, loaded := rpcOzone.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.GetOzoneResult), loaded
	}
	return nil, loaded
}

func SetQueryOzoneResult(key string, result *rpc.GetOzoneResult) {
	if result != nil {
		rpcOzone.Store(key, result)
	}
}
