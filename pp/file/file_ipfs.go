package file

import (
	"strconv"
	"sync"

	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/api/ipfsrpc"
)

var (
	ipfsEventMutex sync.Mutex

	// key(fileReqId) : value(chan *ipfsrpc.DownloadResul)
	ipfsRpcDownloadEventChan = &sync.Map{}

	// key(fileReqId) : value(chan *ipfsrpc.UploadResul)
	ipfsRpcUploadEventChan = &sync.Map{}

	// key(fileReqId) : value(chan *ipfsrpc.FileListResul)
	ipfsRpcFileListEventChan = &sync.Map{}
)

func SetSuccessIpfsDownloadDataResult(key string) {
	SetIpfsDownloadResult(key, ipfsrpc.DownloadResult{Return: ipfsrpc.DOWNLOAD_DATA})
}

func SetSuccessIpfsDownloadFileResult(key string) {
	SetIpfsDownloadResult(key, ipfsrpc.DownloadResult{Return: ipfsrpc.SUCCESS})
}

func SetFailIpfsDownloadResult(key, message string) {
	SetIpfsDownloadResult(key, ipfsrpc.DownloadResult{Return: ipfsrpc.FAILED, Message: message})
}

func SetIpfsDownloadResult(key string, result ipfsrpc.DownloadResult) {
	ipfsEventMutex.Lock()
	defer ipfsEventMutex.Unlock()
	ch, found := ipfsRpcDownloadEventChan.Load(key)
	if found {
		select {
		case ch.(chan *ipfsrpc.DownloadResult) <- &result:
		default:
			ipfsRpcDownloadEventChan.Delete(key)
		}
	}
}

func SetSuccessIpfsUploadDataResult(key string) {
	SetIpfsUploadResult(key, ipfsrpc.UploadResult{Return: ipfsrpc.UPLOAD_DATA})
}

func SetSuccessIpfsUploadFileResult(key string, fileHash string, size int64) {
	SetIpfsUploadResult(key, ipfsrpc.UploadResult{
		Return: ipfsrpc.SUCCESS,
		Hash:   fileHash,
		Bytes:  size,
		Size:   strconv.FormatInt(size, 10)})
}

func SetFailIpfsUploadResult(key, message string) {
	SetIpfsUploadResult(key, ipfsrpc.UploadResult{Return: ipfsrpc.FAILED, Message: message})
}

func SetIpfsUploadResult(key string, result ipfsrpc.UploadResult) {
	ipfsEventMutex.Lock()
	defer ipfsEventMutex.Unlock()
	ch, found := ipfsRpcUploadEventChan.Load(key)
	if found {
		select {
		case ch.(chan *ipfsrpc.UploadResult) <- &result:
		default:
			ipfsRpcUploadEventChan.Delete(key)
		}
	}
}

func SetSuccessIpfsFileListResult(key string, fileList *protos.RspFindMyFileList) {
	_, found := ipfsRpcFileListEventChan.Load(key)
	if !found {
		return
	}

	result := ipfsrpc.FileListResult{Return: ipfsrpc.SUCCESS}
	var fileInfos = make([]ipfsrpc.FileInfo, 0)
	for _, info := range fileList.FileInfo {
		fileInfos = append(fileInfos, ipfsrpc.FileInfo{
			FileHash:   info.FileHash,
			FileSize:   info.FileSize,
			FileName:   info.FileName,
			CreateTime: info.CreateTime,
		})
	}
	result.TotalNumber = fileList.TotalFileNumber
	result.PageId = fileList.PageId
	result.FileInfo = fileInfos

	SetIpfsFileListResult(key, result)
}

func SetIpfsFileListResult(key string, result ipfsrpc.FileListResult) {
	ipfsEventMutex.Lock()
	defer ipfsEventMutex.Unlock()
	ch, found := ipfsRpcFileListEventChan.Load(key)
	if found {
		select {
		case ch.(chan *ipfsrpc.FileListResult) <- &result:
		default:
			ipfsRpcFileListEventChan.Delete(key)
		}
	}
}

func SubscribeIpfsDownload(key string) chan *ipfsrpc.DownloadResult {
	ch := make(chan *ipfsrpc.DownloadResult)
	ipfsRpcDownloadEventChan.Store(key, ch)
	return ch
}

func SubscribeIpfsUpload(key string) chan *ipfsrpc.UploadResult {
	ch := make(chan *ipfsrpc.UploadResult)
	ipfsRpcUploadEventChan.Store(key, ch)
	return ch
}

func SubscribeIpfsFileList(key string) chan *ipfsrpc.FileListResult {
	ch := make(chan *ipfsrpc.FileListResult)
	ipfsRpcFileListEventChan.Store(key, ch)
	return ch
}

func UnsubscribeIpfsDownload(key string) {
	ipfsEventMutex.Lock()
	defer ipfsEventMutex.Unlock()
	ipfsRpcDownloadEventChan.Delete(key)
}

func UnsubscribeIpfsUpload(key string) {
	ipfsEventMutex.Lock()
	defer ipfsEventMutex.Unlock()
	ipfsRpcUploadEventChan.Delete(key)
}

func UnsubscribeIpfsFileList(key string) {
	ipfsEventMutex.Lock()
	defer ipfsEventMutex.Unlock()
	ipfsRpcFileListEventChan.Delete(key)
}
