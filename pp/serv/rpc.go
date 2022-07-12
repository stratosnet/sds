package serv

import (
	"crypto/sha256"
	b64 "encoding/base64"
	"context"
	"encoding/hex"
	"sync"
	"encoding/json"
	"time"
	"github.com/google/uuid"
	"github.com/stratosnet/sds/msg/header"
	rpc_api "github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/rpc"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	"github.com/stratosnet/stratos-chain/types"
	"github.com/tendermint/tendermint/libs/bech32"
)

const (
	// the length of request shall be shorter than 5242880 bytes
	// this equals 3932160 bytes after
	FILE_DATA_SAFE_SIZE = 3500000

	// timeout for waiting result from external source, in seconds
	WAIT_TIMEOUT time.Duration = 5 * time.Second
)

var (
	// key: fileHash value: file
	FileOffset = make(map[string]*FileFetchOffset)
	FileOffsetMutex sync.Mutex
)

type FileFetchOffset struct {
	RemoteRequested    uint64
	ResourceNodeAsked  uint64
}

type rpcApi struct {

}

func RpcApi() *rpcApi {
	return &rpcApi{}
}

// apis returns the collection of built-in RPC APIs.
func apis() []rpc.API {
	return []rpc.API{
		{
			Namespace: "user",
			Version:   "1.0",
			Service:   RpcApi(),
			Public:    true,
		},
	}
}

// ResultHook
func ResultHook(r *rpc_api.Result, fileHash string) *rpc_api.Result {
	if r.Return == rpc_api.UPLOAD_DATA {
		start := *r.OffsetStart
		end := *r.OffsetEnd
		// have to cut the requested data block into smaller pieces when the size is greater than the limit
		if end - start > FILE_DATA_SAFE_SIZE {
			f := &FileFetchOffset{RemoteRequested: start + FILE_DATA_SAFE_SIZE, ResourceNodeAsked: end}

			FileOffsetMutex.Lock()
			FileOffset[fileHash] = f
			FileOffsetMutex.Unlock()

			e := start + FILE_DATA_SAFE_SIZE
			nr := &rpc_api.Result {
				Return: r.Return,
				OffsetStart: &start,
				OffsetEnd: &e,
			}
			return nr
		}
	}
	return r
}

func (api *rpcApi) RequestUpload(param rpc_api.ParamReqUploadFile) rpc_api.Result {
	fileName := param.FileName
	fileSize := param.FileSize
	fileHash := param.FileHash
	walletAddr := param.WalletAddr
	pubkey := param.WalletPubkey
	signature := param.Signature

	size:= fileSize

	// the input for signature is hashed by SHA256
	hs := sha256.Sum256([]byte(fileHash + walletAddr))
	ds, _ := hex.DecodeString(signature)

	// decode public key
	pubPref, pubkey64, err := bech32.DecodeAndConvert(pubkey)
	if pubPref != types.AccountPubKeyPrefix || err != nil {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}
	pk, e := b64.StdEncoding.DecodeString(string(pubkey64))
	if e != nil {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}
	if !secp256k1.VerifySignature(pk, hs[:], ds) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// start to upload file
	p := requests.RequestUploadFile(fileName, fileHash, uint64(size),  "rpc", walletAddr, false)
	peers.SendMessageToSPServer(p, header.ReqUploadFile)

	var result *rpc_api.Result
	var found bool
	var done = make(chan bool)

	go func() {
		for {
			result, found = file.GetRemoteFileEvent(fileHash)
			if result != nil && found {
				result = ResultHook(result, fileHash)
				done <- true
				return
			}
		}
	}()

	select {
	case <-time.After(WAIT_TIMEOUT):
		utils.DebugLog("TO QUIT TIMEOUT")
		return rpc_api.Result{Return: rpc_api.TIME_OUT}
	case <-done:
		mj, _ := json.Marshal(&result)
		utils.DebugLog("Marshal result:", string(mj))

		return *result
	}
}

func (api *rpcApi) UploadData(param rpc_api.ParamUploadData) rpc_api.Result {

	content := param.Data
	fileHash := param.FileHash
	// content in base64
	dec, _ := b64.StdEncoding.DecodeString(content)

	file.SendFileDataBack(fileHash, dec)

	// first part: if the amount of bytes server requested haven't been finished,
	// go on asking from the client
	FileOffsetMutex.Lock()
	fo, found := FileOffset[fileHash]
	FileOffsetMutex.Unlock()
	if found {
		if fo.ResourceNodeAsked - fo.RemoteRequested > FILE_DATA_SAFE_SIZE {
			start := fo.RemoteRequested
			end := fo.RemoteRequested + FILE_DATA_SAFE_SIZE
			nr := rpc_api.Result{
				Return: rpc_api.UPLOAD_DATA,
				OffsetStart: &start,
				OffsetEnd: &end,
			}

			FileOffsetMutex.Lock()
			FileOffset[fileHash].RemoteRequested = fo.RemoteRequested + FILE_DATA_SAFE_SIZE
			FileOffsetMutex.Unlock()
			return nr
		} else {
			nr := rpc_api.Result{
				Return: rpc_api.UPLOAD_DATA,
				OffsetStart: &fo.RemoteRequested,
				OffsetEnd: &fo.ResourceNodeAsked,
			}

			FileOffsetMutex.Lock()
			delete(FileOffset, fileHash)
			FileOffsetMutex.Unlock()
			return nr
		}
	}

	// second part: let the server decide what will be the next step
	var result *rpc_api.Result
	var done = make(chan bool)

	go func() {
		for {
			result, found = file.GetRemoteFileEvent(fileHash)
			if found {
				result = ResultHook(result, fileHash)
				done <- true
				return
			}
		}
	}()

	select {
	case <-time.After(WAIT_TIMEOUT):
		return rpc_api.Result{Return: rpc_api.TIME_OUT}
	case <-done:
		return *result
	}
}

// RequestDownload
func (api *rpcApi) RequestDownload(param rpc_api.ParamReqDownloadFile) rpc_api.Result {

	fileHash := param.FileHash
	wallet := param.WalletAddr

	// request for downloading file
	req, reqid := requests.RequestDownloadFile(fileHash, wallet, "", nil)
	peers.SendMessageDirectToSPOrViaPP(req, header.ReqFileStorageInfo)
	key := fileHash + reqid

	// wait for the result
	var event = make(chan bool)
	var result *rpc_api.Result
	var found bool

	go func() {
		for {
			result, found = file.GetRemoteFileEvent(key)
			if found {
				event <- true
				break
			}
		}
	}()

	select {
	case <-time.After(WAIT_TIMEOUT * 4):
		// end of the session
		file.CleanFileHash(key)
		return rpc_api.Result{Return: rpc_api.TIME_OUT}
	case <-event:
	}

	// one piece to be sent to client
	if result.Return == rpc_api.DOWNLOAD_OK {
		rawData := file.GetDownloadFileData(key)
		utils.DebugLog("Ready sending to Remote:")
		encoded := b64.StdEncoding.EncodeToString(rawData)
		result.FileData = encoded
		result.ReqId = reqid
	} else {
		// end of the session
		file.CleanFileHash(key)
	}

	return *result
}

// DownloadData
func (api *rpcApi) DownloadData(param rpc_api.ParamDownloadData) rpc_api.Result {
	key := param.FileHash + param.ReqId

	// previous piece was done, tell the caller of remote file driver to move on
	file.SetDownloadSliceDone(key)

	// wait for result: DOWNLOAD_OK or DL_OK_ASK_INFO
	var event = make(chan bool)
	var result *rpc_api.Result
	var found bool

	go func() {
		for {
			result, found = file.GetRemoteFileEvent(key)
			if found {
				event <- true
				break
			}
		}
	}()

	// wait too long, failure of timeout
	select {
	case <-time.After(WAIT_TIMEOUT):
		// end of the session
		file.CleanFileHash(key)
		return rpc_api.Result{Return: rpc_api.TIME_OUT}
	case <-event:
	}


	if result.Return == rpc_api.DOWNLOAD_OK {
		rawData := file.GetDownloadFileData(key)
		encoded := b64.StdEncoding.EncodeToString(rawData)
		result.FileData = encoded
	}else if result.Return == rpc_api.DL_OK_ASK_INFO {
		// finished download, and ask the file info to verify downloaded file
	}else {
		// end of the session
		file.CleanFileHash(key)
	}

	return *result

}

// DownloadedFileInfo
func (api *rpcApi) DownloadedFileInfo(param rpc_api.ParamDownloadFileInfo) rpc_api.Result {

	fileSize := param.FileSize
	key := param.FileHash + param.ReqId

	// no matter what reason, this is the end of the session, clean everything related to tthe session
	defer file.CleanFileHash(key)

	file.SetRemoteFileInfo(key, fileSize)

	// wait for result, SUCCESS or some failure
	var result *rpc_api.Result
	var found bool
	var event = make(chan bool)

	go func() {
		for {
			result, found = file.GetRemoteFileEvent(key)
			if found {
				event <- true
				break
			}
		}
	}()

	// wait too long, failure of timeout
	select {
	case <-time.After(WAIT_TIMEOUT):
		return rpc_api.Result{Return: rpc_api.TIME_OUT}
	case <-event:
	}

	return *result
}

// RequestFileList
func (api *rpcApi) RequestList(param rpc_api.ParamReqFileList) rpc_api.FileListResult {

	reqId := uuid.New().String()
	parentCtx := context.Background()
	ctx, _ := context.WithTimeout(parentCtx, WAIT_TIMEOUT)

	event.FindFileList("", param.WalletAddr, param.PageId, reqId, "", 0, true, nil)

	// wait for result, SUCCESS or some failure
	var result *rpc_api.FileListResult
	var found bool

	for {
		select {
		case <-ctx.Done():
			result = &rpc_api.FileListResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found = file.GetFileListResult(param.WalletAddr+reqId)
			if result != nil && found {
				return *result
			}
		}
	}

	return *result
}

// RequestShare
func (api *rpcApi) RequestShare(param rpc_api.ParamReqShareFile) rpc_api.FileShareResult {

	reqId := uuid.New().String()
	parentCtx := context.Background()
	ctx, _ := context.WithTimeout(parentCtx, WAIT_TIMEOUT)

	event.GetReqShareFile(reqId, param.FileHash, "", param.WalletAddr, param.Duration, param.PrivateFlag, nil)

	// wait for result, SUCCESS or some failure
	var result *rpc_api.FileShareResult
	var found bool

	for {
		select {
		case <-ctx.Done():
			result = &rpc_api.FileShareResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found = file.GetFileShareResult(param.WalletAddr+reqId)
			if result != nil && found {
				return *result
			}
		}
	}

	return *result
}

// RequestListShare
func (api *rpcApi) RequestListShare(param rpc_api.ParamReqListShared) rpc_api.FileShareResult {

	reqId := uuid.New().String()
	parentCtx := context.Background()
	ctx, _ := context.WithTimeout(parentCtx, WAIT_TIMEOUT)

	event.GetAllShareLink(reqId, param.WalletAddr, param.PageId, nil)

	// wait for result, SUCCESS or some failure
	var result *rpc_api.FileShareResult
	var found bool

	for {
		select {
		case <-ctx.Done():
			result = &rpc_api.FileShareResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found = file.GetFileShareResult(param.WalletAddr+reqId)
			if result != nil && found {
				return *result
			}
		}
	}

	return *result
}

// RequestStopShare
func (api *rpcApi) RequestStopShare(param rpc_api.ParamReqStopShare) rpc_api.FileShareResult {

	reqId := uuid.New().String()
	parentCtx := context.Background()
	ctx, _ := context.WithTimeout(parentCtx, WAIT_TIMEOUT)

	event.DeleteShare(param.ShareId, reqId, param.WalletAddr, nil)

	// wait for result, SUCCESS or some failure
	var result *rpc_api.FileShareResult
	var found bool

	for {
		select {
		case <-ctx.Done():
			result = &rpc_api.FileShareResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found = file.GetFileShareResult(param.WalletAddr+reqId)
			if result != nil && found {
				return *result
			}
		}
	}

	return *result
}

// RequestGetShared
func (api *rpcApi) RequestGetShared(param rpc_api.ParamReqGetShared) rpc_api.Result {

	reqId := uuid.New().String()
	parentCtx := context.Background()
	ctx, _ := context.WithTimeout(parentCtx, WAIT_TIMEOUT)
	key := param.WalletAddr + reqId

	event.GetShareFile(param.ShareLink, "", "", reqId, param.WalletAddr, nil)

	// the application gives FileShareResult type of result
	var res *rpc_api.FileShareResult

	// only in case of "shared file dl started", jump to next step. Otherwise, return.
	found := false
	for !found {
		select {
		case <-ctx.Done():
			return rpc_api.Result{Return: rpc_api.TIME_OUT}
		default:
			res, found = file.GetFileShareResult(param.WalletAddr + reqId)
			if found {
				// the result is read, but it's nil
				if res == nil {
					return rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE}
				}
				// if shared download has started, wait for the rsp of file storage info
				if res.Return != rpc_api.SHARED_DL_START {
					return rpc_api.Result{Return: res.Return}
				}
			}
		}
	}

	// file hash should be given in the result message
	fileHash := res.FileInfo[0].FileHash
	if fileHash == "" {
		return rpc_api.Result{Return: rpc_api.WRONG_FILE_INFO}
	}

	// start from here, the control flow follows that of download file
	key = fileHash + reqId

	var result *rpc_api.Result
	ctx, _ = context.WithTimeout(parentCtx, WAIT_TIMEOUT)

	found = false
	for !found {
		select {
		case <-ctx.Done():
			file.CleanFileHash(key)
			return rpc_api.Result{Return: rpc_api.TIME_OUT}
		default:
			result, found = file.GetRemoteFileEvent(key)
		}
	}

	// one piece to be sent to client
	if result.Return == rpc_api.DOWNLOAD_OK {
		rawData := file.GetDownloadFileData(key)
		encoded := b64.StdEncoding.EncodeToString(rawData)
		result.FileData = encoded
		result.ReqId = reqId
	} else {
		// end of the session
		file.CleanFileHash(key)
	}

	return *result
}
