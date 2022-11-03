package serv

import (
	"context"
	b64 "encoding/base64"
	"encoding/hex"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stratosnet/sds/metrics"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	rpc_api "github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/rpc"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/datamesh"
	utiltypes "github.com/stratosnet/sds/utils/types"
)

const (
	// FILE_DATA_SAFE_SIZE the length of request shall be shorter than 5242880 bytes
	// this equals 3932160 bytes after
	FILE_DATA_SAFE_SIZE = 3500000

	// WAIT_TIMEOUT timeout for waiting result from external source, in seconds
	WAIT_TIMEOUT time.Duration = 10 * time.Second

	// INIT_WAIT_TIMEOUT timeout for waiting the initial request
	INIT_WAIT_TIMEOUT time.Duration = 15 * time.Second
)

var (
	// key: fileHash value: file
	FileOffset      = make(map[string]*FileFetchOffset)
	FileOffsetMutex sync.Mutex
)

type FileFetchOffset struct {
	RemoteRequested   uint64
	ResourceNodeAsked uint64
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
		if end-start > FILE_DATA_SAFE_SIZE {
			f := &FileFetchOffset{RemoteRequested: start + FILE_DATA_SAFE_SIZE, ResourceNodeAsked: end}

			FileOffsetMutex.Lock()
			FileOffset[fileHash] = f
			FileOffsetMutex.Unlock()

			e := start + FILE_DATA_SAFE_SIZE
			nr := &rpc_api.Result{
				Return:      r.Return,
				OffsetStart: &start,
				OffsetEnd:   &e,
			}
			return nr
		}
	}
	return r
}

func (api *rpcApi) RequestUpload(param rpc_api.ParamReqUploadFile) rpc_api.Result {
	metrics.RpcReqCount.WithLabelValues("RequestUpload").Inc()
	fileName := param.FileName
	fileSize := param.FileSize
	fileHash := param.FileHash
	walletAddr := param.WalletAddr
	pubkey := param.WalletPubkey
	signature := param.Signature
	size := fileSize

	// verify if wallet and public key match
	if utiltypes.VerifyWalletAddr(pubkey, walletAddr) != 0 {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}
	// verify the signature
	if !utiltypes.VerifyWalletSign(pubkey, signature, utils.GetFileUploadWalletSignMessage(fileHash, walletAddr)) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// start to upload file
	p := requests.RequestUploadFile(fileName, fileHash, uint64(size), walletAddr, pubkey, signature, false)
	peers.SendMessageToSPServer(context.Background(), p, header.ReqUploadFile)

	//var done = make(chan bool)
	parentCtx := context.Background()
	ctx, _ := context.WithTimeout(parentCtx, INIT_WAIT_TIMEOUT)

	select {
	case <-ctx.Done():
		result := &rpc_api.Result{Return: rpc_api.TIME_OUT}
		return *result
	// since request for uploading a file has been invoked, wait for application's reply then return the result back to the rpc client
	case result := <-file.SubscribeRemoteFileEvent(fileHash):
		file.UnsubscribeRemoteFileEvent(fileHash)
		if result != nil {
			result = ResultHook(result, fileHash)
			return *result
		} else {
			result = &rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE}
			return *result
		}
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
		if fo.ResourceNodeAsked-fo.RemoteRequested > FILE_DATA_SAFE_SIZE {
			start := fo.RemoteRequested
			end := fo.RemoteRequested + FILE_DATA_SAFE_SIZE
			nr := rpc_api.Result{
				Return:      rpc_api.UPLOAD_DATA,
				OffsetStart: &start,
				OffsetEnd:   &end,
			}

			FileOffsetMutex.Lock()
			FileOffset[fileHash].RemoteRequested = fo.RemoteRequested + FILE_DATA_SAFE_SIZE
			FileOffsetMutex.Unlock()
			return nr
		} else {
			nr := rpc_api.Result{
				Return:      rpc_api.UPLOAD_DATA,
				OffsetStart: &fo.RemoteRequested,
				OffsetEnd:   &fo.ResourceNodeAsked,
			}

			FileOffsetMutex.Lock()
			delete(FileOffset, fileHash)
			FileOffsetMutex.Unlock()
			return nr
		}
	}

	// second part: let the server decide what will be the next step
	parentCtx := context.Background()
	ctx, _ := context.WithTimeout(parentCtx, WAIT_TIMEOUT)

	select {
	case <-ctx.Done():
		result := &rpc_api.Result{Return: rpc_api.TIME_OUT}
		return *result
	// since a slice has been passed to the application, wait for application's reply then return the result back to the rpc client
	case result := <-file.SubscribeRemoteFileEvent(fileHash):
		file.UnsubscribeRemoteFileEvent(fileHash)
		if result != nil {
			result = ResultHook(result, fileHash)
			return *result
		} else {
			result = &rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE}
			return *result
		}
	}
}

// RequestDownload
func (api *rpcApi) RequestDownload(param rpc_api.ParamReqDownloadFile) rpc_api.Result {
	metrics.RpcReqCount.WithLabelValues("RequestDownload").Inc()
	_, _, fileHash, _, err := datamesh.ParseFileHandle(param.FileHandle)
	if err != nil {
		return rpc_api.Result{Return: rpc_api.WRONG_INPUT}
	}

	wallet := param.WalletAddr
	pubkey := param.WalletPubkey
	signature := param.Signature

	// wallet pubkey and wallet signature will be carried in sds messages in []byte format
	wpk, err := utiltypes.WalletPubkeyFromBech(pubkey)
	if err != nil {
		utils.ErrorLog("wrong wallet pubkey")
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}
	wsig, err := hex.DecodeString(signature)
	if err != nil {
		utils.ErrorLog("wrong signature")
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// verify if wallet and public key match
	if utiltypes.VerifyWalletAddrBytes(wpk.Bytes(), wallet) != 0 {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// verify the signature
	wsigMsg := utils.GetFileDownloadWalletSignMessage(fileHash, wallet)
	if !utiltypes.VerifyWalletSignBytes(wpk.Bytes(), wsig, wsigMsg) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	reqId := uuid.New().String()
	reqCtx := core.RegisterRemoteReqId(context.Background(), reqId)
	// request for downloading file
	req := requests.RequestDownloadFile(fileHash, param.FileHandle, wallet, reqId, wsig, wpk.Bytes(), nil)
	peers.SendMessageDirectToSPOrViaPP(reqCtx, req, header.ReqFileStorageInfo)
	key := fileHash + reqId

	// wait for the result
	parentCtx := context.Background()
	ctx, _ := context.WithTimeout(parentCtx, WAIT_TIMEOUT)
	var result *rpc_api.Result

	select {
	case <-ctx.Done():
		file.CleanFileHash(key)
		result = &rpc_api.Result{Return: rpc_api.TIME_OUT}
	// since downloading a file has been requested, wait for application's reply then return the result back to the rpc client
	case result = <-file.SubscribeRemoteFileEvent(key):
		file.UnsubscribeRemoteFileEvent(key)
		// one piece to be sent to client
		if result != nil && result.Return == rpc_api.DOWNLOAD_OK {
			result.ReqId = reqId
		} else {
			// end of the session
			file.CleanFileHash(key)
		}
	}

	return *result
}

// DownloadData
func (api *rpcApi) DownloadData(param rpc_api.ParamDownloadData) rpc_api.Result {
	key := param.FileHash + param.ReqId

	// previous piece was done, tell the caller of remote file driver to move on
	file.SetDownloadSliceDone(key)

	// wait for result: DOWNLOAD_OK or DL_OK_ASK_INFO
	parentCtx := context.Background()
	ctx, _ := context.WithTimeout(parentCtx, WAIT_TIMEOUT)
	var result *rpc_api.Result

	select {
	case <-ctx.Done():
		file.CleanFileHash(key)
		result = &rpc_api.Result{Return: rpc_api.TIME_OUT}
	// told application that last piece has been done, wait here for the next piece or other event and send this back to rpc client
	case result = <-file.SubscribeRemoteFileEvent(key):
		file.UnsubscribeRemoteFileEvent(key)
		if result == nil || !(result.Return == rpc_api.DOWNLOAD_OK || result.Return == rpc_api.DL_OK_ASK_INFO) {
			file.CleanFileHash(key)
		}
	}
	return *result
}

// DownloadedFileInfo
func (api *rpcApi) DownloadedFileInfo(param rpc_api.ParamDownloadFileInfo) rpc_api.Result {
	metrics.RpcReqCount.WithLabelValues("DownloadedFileInfo").Inc()

	fileSize := param.FileSize
	key := param.FileHash + param.ReqId

	// no matter what reason, this is the end of the session, clean everything related to tthe session
	defer file.CleanFileHash(key)

	file.SetRemoteFileInfo(key, fileSize)

	// wait for result, SUCCESS or some failure
	parentCtx := context.Background()
	ctx, _ := context.WithTimeout(parentCtx, WAIT_TIMEOUT)
	var result *rpc_api.Result

	select {
	case <-ctx.Done():
		file.CleanFileHash(key)
		result = &rpc_api.Result{Return: rpc_api.TIME_OUT}
	// the file info at the end has been sent to the application, wait to confirm the end of download process and send this
	// back to rpc client.
	case result = <-file.SubscribeRemoteFileEvent(key):
		file.UnsubscribeRemoteFileEvent(key)
	}

	return *result
}

// RequestList
func (api *rpcApi) RequestList(param rpc_api.ParamReqFileList) rpc_api.FileListResult {
	metrics.RpcReqCount.WithLabelValues("RequestList").Inc()

	reqId := uuid.New().String()
	parentCtx := context.Background()
	ctx, _ := context.WithTimeout(parentCtx, WAIT_TIMEOUT)
	reqCtx := core.RegisterRemoteReqId(context.Background(), reqId)
	event.FindFileList(reqCtx, "", param.WalletAddr, param.PageId, "", 0, true, nil)

	// wait for result, SUCCESS or some failure
	var result *rpc_api.FileListResult
	var found bool

	for {
		select {
		case <-ctx.Done():
			result = &rpc_api.FileListResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found = file.GetFileListResult(param.WalletAddr + reqId)
			if result != nil && found {
				return *result
			}
		}
	}

	return *result
}

// RequestShare
func (api *rpcApi) RequestShare(param rpc_api.ParamReqShareFile) rpc_api.FileShareResult {
	metrics.RpcReqCount.WithLabelValues("RequestShare").Inc()
	reqId := uuid.New().String()
	parentCtx := context.Background()
	ctx, _ := context.WithTimeout(parentCtx, WAIT_TIMEOUT)
	reqCtx := core.RegisterRemoteReqId(context.Background(), reqId)
	event.GetReqShareFile(reqCtx, param.FileHash, "", param.WalletAddr, param.Duration, param.PrivateFlag, nil)

	// wait for result, SUCCESS or some failure
	var result *rpc_api.FileShareResult
	var found bool

	for {
		select {
		case <-ctx.Done():
			result = &rpc_api.FileShareResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found = file.GetFileShareResult(param.WalletAddr + reqId)
			if result != nil && found {
				return *result
			}
		}
	}

	return *result
}

// RequestListShare
func (api *rpcApi) RequestListShare(param rpc_api.ParamReqListShared) rpc_api.FileShareResult {
	metrics.RpcReqCount.WithLabelValues("RequestListShare").Inc()
	reqId := uuid.New().String()
	parentCtx := context.Background()
	ctx, _ := context.WithTimeout(parentCtx, WAIT_TIMEOUT)
	reqCtx := core.RegisterRemoteReqId(context.Background(), reqId)
	event.GetAllShareLink(reqCtx, param.WalletAddr, param.PageId, nil)

	// wait for result, SUCCESS or some failure
	var result *rpc_api.FileShareResult
	var found bool

	for {
		select {
		case <-ctx.Done():
			result = &rpc_api.FileShareResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found = file.GetFileShareResult(param.WalletAddr + reqId)
			if result != nil && found {
				return *result
			}
		}
	}

	return *result
}

// RequestStopShare
func (api *rpcApi) RequestStopShare(param rpc_api.ParamReqStopShare) rpc_api.FileShareResult {
	metrics.RpcReqCount.WithLabelValues("RequestStopShare").Inc()
	reqId := uuid.New().String()
	parentCtx := context.Background()
	ctx, _ := context.WithTimeout(parentCtx, WAIT_TIMEOUT)
	reqCtx := core.RegisterRemoteReqId(context.Background(), reqId)
	event.DeleteShare(reqCtx, param.ShareId, param.WalletAddr, nil)

	// wait for result, SUCCESS or some failure
	var result *rpc_api.FileShareResult
	var found bool

	for {
		select {
		case <-ctx.Done():
			result = &rpc_api.FileShareResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found = file.GetFileShareResult(param.WalletAddr + reqId)
			if result != nil && found {
				return *result
			}
		}
	}

	return *result
}

// RequestGetShared
func (api *rpcApi) RequestGetShared(param rpc_api.ParamReqGetShared) rpc_api.Result {
	metrics.RpcReqCount.WithLabelValues("RequestGetShared").Inc()
	wallet := param.WalletAddr
	pubkey := param.WalletPubkey
	signature := param.Signature

	// wallet pubkey and wallet signature will be carried in sds messages in []byte format
	wpk, err := utiltypes.WalletPubkeyFromBech(pubkey)
	if err != nil {
		utils.ErrorLog("wrong wallet pubkey")
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}
	wsig, err := hex.DecodeString(signature)
	if err != nil {
		utils.ErrorLog("wrong signature")
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// verify if wallet and public key match
	if utiltypes.VerifyWalletAddrBytes(wpk.Bytes(), wallet) != 0 {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// verify the signature
	wsigMsg := utils.GetFileDownloadShareWalletSignMessage(param.FileHash, wallet)
	if !utiltypes.VerifyWalletSignBytes(wpk.Bytes(), wsig, wsigMsg) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	reqId := uuid.New().String()
	parentCtx := context.Background()
	ctx, _ := context.WithTimeout(parentCtx, WAIT_TIMEOUT)
	key := param.WalletAddr + reqId

	reqCtx := core.RegisterRemoteReqId(context.Background(), reqId)
	event.GetShareFile(reqCtx, param.ShareLink, "", "", param.WalletAddr, wpk.Bytes(), wsig, nil)

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

	select {
	case <-ctx.Done():
		file.CleanFileHash(key)
		return rpc_api.Result{Return: rpc_api.TIME_OUT}
	case result = <-file.SubscribeRemoteFileEvent(key):
		file.UnsubscribeRemoteFileEvent(key)
	}

	// one piece to be sent to client
	if result.Return == rpc_api.DOWNLOAD_OK {
		result.ReqId = reqId
	} else {
		// end of the session
		file.CleanFileHash(key)
	}

	return *result
}

// RequestGetOzone
func (api *rpcApi) RequestGetOzone(param rpc_api.ParamReqGetOzone) rpc_api.GetOzoneResult {
	metrics.RpcReqCount.WithLabelValues("RequestGetOzone").Inc()
	reqId := uuid.New().String()
	parentCtx := context.Background()
	ctx, _ := context.WithTimeout(parentCtx, WAIT_TIMEOUT)
	err := event.GetWalletOz(core.RegisterRemoteReqId(context.Background(), reqId), param.WalletAddr, reqId)
	if err != nil {
		return rpc_api.GetOzoneResult{Return: rpc_api.TIME_OUT}
	}

	// wait for result, SUCCESS or some failure
	var result *rpc_api.GetOzoneResult
	var found bool

	for {
		select {
		case <-ctx.Done():
			result = &rpc_api.GetOzoneResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found = file.GetQueryOzoneResult(param.WalletAddr + reqId)
			if result != nil && found {
				return *result
			}
		}
	}

	return *result
}
