package file

import (
	"context"
	"strings"
	"sync"

	"github.com/stratosnet/sds/pp/api/ipfsrpc"
	"github.com/stratosnet/sds/pp/api/rpc"
)

var (
	ipfsFileMutex sync.Mutex
	//
	//upSliceMutex sync.Mutex

	ipfsEventMutex sync.Mutex

	// key(fileHash + fileReqId) : value(fileSize)
	//rpcFileInfoMap = &sync.Map{}

	// key(fileHash + fileReqId) : value(chan *rpc.Result)
	ipfsRpcDownloadEventChan = &sync.Map{}

	// key(fileHash + fileReqId) : value(chan *rpc.Result)
	ipfsRpcUploadEventChan = &sync.Map{}

	//// key(fileHash) : value(chan []byte)
	//rpcUploadDataChan = &sync.Map{}
	//
	//// key(fileHash) : value(chan string)
	//rpcSignatureChan = &sync.Map{}
	//
	//// key(fileHash + file) : value(downloadReady)
	//rpcDownloadReady = &sync.Map{}

	// key(fileHash + file) : value(chan uint64)
	//rpcDownloadFileInfo = &sync.Map{}

	// wait for the next request from client per message
	//RpcWaitTimeout time.Duration
)

// GetRemoteFileInfo
func GetIpfsFileDownload(key, reqId string) uint64 {
	SetRemoteFileResult(key, rpc.Result{ReqId: reqId, Return: rpc.DL_OK_ASK_INFO})
	var fileSize uint64
	parentCtx := context.Background()
	ctx, _ := context.WithTimeout(parentCtx, RpcWaitTimeout)

	select {
	case <-ctx.Done():
		return 0
	case fileSize = <-SubscribeDownloadFileInfo(key):
		UnsubscribeDownloadFileInfo(key)
	}
	return fileSize
}

func IsIpfsRpc(key string) bool {
	str := ipfsFileMap[key]
	if str == "" {
		return false
	}
	return strings.Split(str, ":")[0] == "ipfs"
}

func SetSuccessIpfsDownloadResult(key string) {
	SetIpfsDownloadResult(key, ipfsrpc.DownloadResult{Return: ipfsrpc.SUCCESS})
}

func SetFailIpfsDownloadResult(key, message string) {
	SetIpfsDownloadResult(key, ipfsrpc.DownloadResult{Return: ipfsrpc.FAILED, Message: message})
}

// SetIpfsFileDownloadResult
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

func SaveIpfsRemoteFileHash(hash, fileName string) {
	ipfsFileMutex.Lock()
	defer ipfsFileMutex.Unlock()

	ipfsFileMap[hash] = "ipfs:" + fileName
}

//// SetRemoteFileInfo
//func SetIpfsDownload(key string, size uint64) {
//	ipfsEventMutex.Lock()
//	defer ipfsEventMutex.Unlock()
//	ch, found := rpcDownloadFileInfo.Load(key)
//	if found {
//		select {
//		case ch.(chan uint64) <- size:
//		default:
//			rpcDownloadFileInfo.Delete(key)
//		}
//	}
//}
