package file

import (
	"sync"

	"github.com/stratosnet/sds/pp/api/ipfsrpc"
)

var (
	ipfsEventMutex sync.Mutex

	// key(fileReqId) : value(chan *rpc.Result)
	ipfsRpcDownloadEventChan = &sync.Map{}

	// key(fileHash + fileReqId) : value(chan *rpc.Result)
	ipfsRpcUploadEventChan = &sync.Map{}
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
	SetIpfsDownloadResult(key, ipfsrpc.DownloadResult{Return: ipfsrpc.UPLOAD_DATA})
}

func SetSuccessIpfsUploadFileResult(key string) {
	SetIpfsUploadResult(key, ipfsrpc.UploadResult{Return: ipfsrpc.SUCCESS})
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
