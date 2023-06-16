package namespace

import (
	"context"
	b64 "encoding/base64"
	"encoding/hex"
	"math"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/metrics"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	rpc_api "github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
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

type rpcPubApi struct {
}

func RpcPubApi() *rpcPubApi {
	return &rpcPubApi{}
}

type rpcPrivApi struct {
}

func RpcPrivApi() *rpcPrivApi {
	return &rpcPrivApi{}
}

// apis returns the collection of built-in RPC APIs.
func Apis() []rpc.API {
	return []rpc.API{
		{
			Namespace: "owner",
			Version:   "1.0",
			Service:   RpcPrivApi(),
			Public:    false,
		},
		{
			Namespace: "user",
			Version:   "1.0",
			Service:   RpcPubApi(),
			Public:    true,
		},
	}
}

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

func (api *rpcPubApi) RequestUpload(ctx context.Context, param rpc_api.ParamReqUploadFile) rpc_api.Result {
	metrics.RpcReqCount.WithLabelValues("RequestUpload").Inc()
	fileHash := param.FileHash
	walletAddr := param.WalletAddr
	pubkey := param.WalletPubkey

	// verify if wallet and public key match
	if utiltypes.VerifyWalletAddr(pubkey, walletAddr) != 0 {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// fetch file slices from remote client and send upload request to sp
	fetchRemoteFileAndReqUpload := func() {
		metrics.UploadPerformanceLogNow(param.FileHash + ":RCV_REQ_UPLOAD_CLIENT")
		fileName := param.FileName
		fileSize := uint64(param.FileSize)
		signature := param.Signature

		sliceSize := uint64(setting.DEFAULT_SLICE_BLOCK_SIZE)
		sliceCount := uint64(math.Ceil(float64(fileSize) / float64(sliceSize)))

		var slices []*protos.SliceHashAddr
		for sliceNumber := uint64(1); sliceNumber <= sliceCount; sliceNumber++ {
			sliceOffset := requests.GetSliceOffset(sliceNumber, sliceCount, sliceSize, fileSize)

			tmpSliceName := uuid.NewString()
			var rawData []byte
			var err error
			if file.CacheRemoteFileData(fileHash, sliceOffset, tmpSliceName) == nil {
				rawData, err = file.GetSliceDataFromTmp(fileHash, tmpSliceName)
				if err != nil {
					file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE})
					return
				}
			}

			//Encrypt slice data if required
			sliceHash := utils.CalcSliceHash(rawData, fileHash, sliceNumber)

			SliceHashAddr := &protos.SliceHashAddr{
				SliceHash:   sliceHash,
				SliceSize:   sliceOffset.SliceOffsetEnd - sliceOffset.SliceOffsetStart,
				SliceNumber: sliceNumber,
				SliceOffset: sliceOffset,
			}

			slices = append(slices, SliceHashAddr)

			err = file.RenameTmpFile(fileHash, tmpSliceName, sliceHash)
			if err != nil {
				file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE})
				return
			}
		}

		// start to upload file
		p, err := requests.RequestUploadFile(ctx, fileName, fileHash, fileSize, walletAddr, pubkey, signature, slices, false, false, param.DesiredTier, param.AllowHigherTier)
		if err != nil {
			file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE})
			return
		}
		metrics.UploadPerformanceLogNow(param.FileHash + ":SND_REQ_UPLOAD_SP")
		p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, p, header.ReqUploadFile)

		defer metrics.UploadPerformanceLogNow(param.FileHash + ":SND_RSP_UPLOAD_CLIENT")
	}

	//var done = make(chan bool)
	ctx, cancel := context.WithTimeout(ctx, INIT_WAIT_TIMEOUT)
	defer cancel()

	fileEventCh := file.SubscribeRemoteFileEvent(fileHash)
	go fetchRemoteFileAndReqUpload()

	select {
	case <-ctx.Done():
		result := &rpc_api.Result{Return: rpc_api.TIME_OUT}
		return *result
	// since request for uploading a file has been invoked, wait for application's reply then return the result back to the rpc client
	case result := <-fileEventCh:
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

func (api *rpcPubApi) UploadData(ctx context.Context, param rpc_api.ParamUploadData) rpc_api.Result {

	metrics.UploadPerformanceLogNow(param.FileHash + ":RCV_REQ_UPLOAD_SP:")

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
	newctx, cancel := context.WithTimeout(context.Background(), WAIT_TIMEOUT)
	defer cancel()

	select {
	case <-newctx.Done():
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

func (api *rpcPubApi) RequestUploadStream(ctx context.Context, param rpc_api.ParamReqUploadFile) rpc_api.Result {
	//metrics.RpcReqCount.WithLabelValues("RequestUpload").Inc()
	//metrics.UploadPerformanceLogNow(param.FileHash + ":RCV_REQ_UPLOAD_CLIENT")
	fileName := param.FileName
	fileSize := param.FileSize
	fileHash := param.FileHash
	walletAddr := param.WalletAddr
	pubkey := param.WalletPubkey
	signature := param.Signature
	size := fileSize

	_, name := filepath.Split(fileName)
	if len(name) > 0 {
		utils.DebugLogf("fileName is trimmed from %v to %v", fileName, name)
		fileName = name
	}

	// verify if wallet and public key match
	if utiltypes.VerifyWalletAddr(pubkey, walletAddr) != 0 {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// start to upload file
	go uploadStreamTmpFile(ctx, fileHash, fileName, uint64(size), walletAddr, pubkey, signature, param.DesiredTier, param.AllowHigherTier)

	ctx, cancel := context.WithTimeout(ctx, INIT_WAIT_TIMEOUT)
	defer cancel()

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

func (api *rpcPubApi) UploadDataStream(ctx context.Context, param rpc_api.ParamUploadData) rpc_api.Result {

	//metrics.UploadPerformanceLogNow(param.FileHash + ":RCV_REQ_UPLOAD_SP:")

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
	newctx, cancel := context.WithTimeout(context.Background(), 3*WAIT_TIMEOUT)
	defer cancel()

	select {
	case <-newctx.Done():
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

func (api *rpcPubApi) RequestDownload(ctx context.Context, param rpc_api.ParamReqDownloadFile) rpc_api.Result {
	metrics.RpcReqCount.WithLabelValues("RequestDownload").Inc()
	_, _, fileHash, _, err := datamesh.ParseFileHandle(param.FileHandle)
	if err != nil {
		return rpc_api.Result{Return: rpc_api.WRONG_INPUT}
	}

	metrics.UploadPerformanceLogNow(fileHash + ":RCV_REQ_DOWNLOAD_CLIENT")
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

	reqId := uuid.New().String()
	ctx = core.RegisterRemoteReqId(ctx, reqId)
	// request for downloading file
	req := requests.RequestDownloadFile(ctx, fileHash, param.FileHandle, wallet, reqId, wsig, wpk.Bytes(), nil)
	p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, req, header.ReqFileStorageInfo)

	key := fileHash + reqId

	// wait for the result
	ctx, cancel := context.WithTimeout(ctx, WAIT_TIMEOUT)
	defer cancel()

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

func (api *rpcPubApi) RequestDownloadSliceData(ctx context.Context, param rpc_api.ParamReqDownloadData) rpc_api.Result {
	ctx = core.RegisterRemoteReqId(ctx, param.ReqId)
	var fInfo *protos.RspFileStorageInfo
	if f, ok := task.DownloadFileMap.Load(param.FileHash + param.ReqId); ok {
		fInfo = f.(*protos.RspFileStorageInfo)
	} else {
		return rpc_api.Result{Return: rpc_api.WRONG_INPUT}
	}

	req := &protos.ReqDownloadSlice{
		RspFileStorageInfo: fInfo,
		SliceNumber:        param.SliceNumber,
		P2PAddress:         p2pserver.GetP2pServer(ctx).GetP2PAddress(),
	}
	networkAddress := param.NetworkAddress
	msgKey := "download#" + param.FileHash + strconv.FormatUint(param.SliceNumber, 10) + param.P2PAddress + param.ReqId
	err := p2pserver.GetP2pServer(ctx).SendMessageByCachedConn(ctx, msgKey, networkAddress, req, header.ReqDownloadSlice, nil)
	if err != nil {
		return rpc_api.Result{Return: rpc_api.INTERNAL_COMM_FAILURE}
	}

	key := param.SliceHash + param.ReqId

	// wait for result: DOWNLOAD_OK
	ctx, cancel := context.WithTimeout(ctx, WAIT_TIMEOUT)
	defer cancel()
	var result *rpc_api.Result

	select {
	case <-ctx.Done():
		result = &rpc_api.Result{Return: rpc_api.TIME_OUT}
	case result = <-file.SubscribeRemoteSliceEvent(key):
		file.UnsubscribeRemoteSliceEvent(key)
	}

	return *result
}

func (api *rpcPubApi) DownloadData(ctx context.Context, param rpc_api.ParamDownloadData) rpc_api.Result {
	key := param.FileHash + param.ReqId

	// previous piece was done, tell the caller of remote file driver to move on
	file.SetDownloadSliceDone(key)

	// wait for result: DOWNLOAD_OK or DL_OK_ASK_INFO
	ctx, cancel := context.WithTimeout(ctx, WAIT_TIMEOUT)
	defer cancel()
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
		if result != nil {
			result.FileName = ""
		}
	}

	return *result
}

func (api *rpcPubApi) DownloadedFileInfo(ctx context.Context, param rpc_api.ParamDownloadFileInfo) rpc_api.Result {
	metrics.RpcReqCount.WithLabelValues("DownloadedFileInfo").Inc()

	fileSize := param.FileSize
	key := param.FileHash + param.ReqId

	// no matter what reason, this is the end of the session, clean everything related to tthe session
	defer file.CleanFileHash(key)

	file.SetRemoteFileInfo(key, fileSize)

	// wait for result, SUCCESS or some failure
	ctx, cancel := context.WithTimeout(ctx, WAIT_TIMEOUT)
	defer cancel()
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

func (api *rpcPubApi) RequestList(ctx context.Context, param rpc_api.ParamReqFileList) rpc_api.FileListResult {
	metrics.RpcReqCount.WithLabelValues("RequestList").Inc()

	reqId := uuid.New().String()
	ctx, cancel := context.WithTimeout(ctx, WAIT_TIMEOUT)
	defer cancel()
	ctx = core.RegisterRemoteReqId(ctx, reqId)
	event.FindFileList(ctx, "", param.WalletAddr, param.PageId, "", 0, true)

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
}

func (api *rpcPubApi) RequestShare(ctx context.Context, param rpc_api.ParamReqShareFile) rpc_api.FileShareResult {
	metrics.RpcReqCount.WithLabelValues("RequestShare").Inc()
	reqId := uuid.New().String()
	ctx, cancel := context.WithTimeout(ctx, WAIT_TIMEOUT)
	defer cancel()
	reqCtx := core.RegisterRemoteReqId(ctx, reqId)
	event.GetReqShareFile(reqCtx, param.FileHash, "", param.WalletAddr, param.Duration, param.PrivateFlag)

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
}

func (api *rpcPubApi) RequestListShare(ctx context.Context, param rpc_api.ParamReqListShared) rpc_api.FileShareResult {
	metrics.RpcReqCount.WithLabelValues("RequestListShare").Inc()
	reqId := uuid.New().String()
	ctx, cancel := context.WithTimeout(ctx, WAIT_TIMEOUT)
	defer cancel()
	reqCtx := core.RegisterRemoteReqId(ctx, reqId)
	event.GetAllShareLink(reqCtx, param.WalletAddr, param.PageId)

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
}

func (api *rpcPubApi) RequestStopShare(ctx context.Context, param rpc_api.ParamReqStopShare) rpc_api.FileShareResult {
	metrics.RpcReqCount.WithLabelValues("RequestStopShare").Inc()
	reqId := uuid.New().String()
	ctx, cancel := context.WithTimeout(ctx, WAIT_TIMEOUT)
	defer cancel()
	reqCtx := core.RegisterRemoteReqId(ctx, reqId)
	event.DeleteShare(reqCtx, param.ShareId, param.WalletAddr)

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
}

func (api *rpcPubApi) RequestGetShared(ctx context.Context, param rpc_api.ParamReqGetShared) rpc_api.Result {
	metrics.RpcReqCount.WithLabelValues("RequestGetShared").Inc()
	wallet := param.WalletAddr
	pubkey := param.WalletPubkey

	// wallet pubkey and wallet signature will be carried in sds messages in []byte format
	wpk, err := utiltypes.WalletPubkeyFromBech(pubkey)
	if err != nil {
		utils.ErrorLog("wrong wallet pubkey")
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// verify if wallet and public key match
	if utiltypes.VerifyWalletAddrBytes(wpk.Bytes(), wallet) != 0 {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	reqId := uuid.New().String()
	ctx, cancel := context.WithTimeout(ctx, WAIT_TIMEOUT)
	defer cancel()
	key := param.WalletAddr + reqId

	reqCtx := core.RegisterRemoteReqId(ctx, reqId)
	event.GetShareFile(reqCtx, param.ShareLink, "", "", param.WalletAddr, wpk.Bytes(), false)

	// the application gives FileShareResult type of result
	var res *rpc_api.FileShareResult

	// only in case of "shared file dl started", jump to next step. Otherwise, return.
	found := false
	for !found {
		select {
		case <-ctx.Done():
			return rpc_api.Result{Return: rpc_api.TIME_OUT}
		default:
			res, found = file.GetFileShareResult(key)
			if found {
				// the result is read, but it's nil
				if res == nil {
					return rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE}
				} else {
					return rpc_api.Result{
						Return:         res.Return,
						ReqId:          reqId,
						FileHash:       res.FileInfo[0].FileHash,
						SequenceNumber: res.SequenceNumber,
					}
				}
			}
		}
	}
	return rpc_api.Result{Return: rpc_api.TIME_OUT}
}

func (api *rpcPubApi) RequestDownloadShared(ctx context.Context, param rpc_api.ParamReqDownloadShared) rpc_api.Result {
	// wallet pubkey and wallet signature will be carried in sds messages in []byte format
	wpk, err := utiltypes.WalletPubkeyFromBech(param.WalletPubkey)
	if err != nil {
		utils.ErrorLog("wrong wallet pubkey")
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// verify if wallet and public key match
	if utiltypes.VerifyWalletAddrBytes(wpk.Bytes(), param.WalletAddr) != 0 {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	wsig, err := hex.DecodeString(param.Signature)
	if err != nil {
		utils.ErrorLog("wrong signature")
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// file hash should be given in the result message
	fileHash := param.FileHash
	if fileHash == "" {
		return rpc_api.Result{Return: rpc_api.WRONG_FILE_INFO}
	}

	file.SetSignature(param.FileHash, wsig)

	// start from here, the control flow follows that of download file
	key := fileHash + param.ReqId

	var result *rpc_api.Result
	ctx, cancel := context.WithTimeout(ctx, WAIT_TIMEOUT)
	defer cancel()

	select {
	case <-ctx.Done():
		file.CleanFileHash(key)
		return rpc_api.Result{Return: rpc_api.TIME_OUT}
	case result = <-file.SubscribeRemoteFileEvent(key):
		file.UnsubscribeRemoteFileEvent(key)
	}

	// one piece to be sent to client
	if result.Return == rpc_api.DOWNLOAD_OK {
		result.ReqId = param.ReqId
	} else {
		// end of the session
		file.CleanFileHash(key)
	}

	return *result
}

func (api *rpcPubApi) RequestGetOzone(ctx context.Context, param rpc_api.ParamReqGetOzone) rpc_api.GetOzoneResult {
	metrics.RpcReqCount.WithLabelValues("RequestGetOzone").Inc()
	reqId := uuid.New().String()
	ctx, cancel := context.WithTimeout(ctx, WAIT_TIMEOUT)
	defer cancel()
	err := event.GetWalletOz(core.RegisterRemoteReqId(ctx, reqId), param.WalletAddr, reqId)
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
}

func uploadStreamTmpFile(ctx context.Context, fileHash, fileName string, fileSize uint64, walletAddr, pubkey, signature string, desiredTier uint32, allowHigherTier bool) {
	if err := file.CacheRemoteFileData(fileHash, &protos.SliceOffset{SliceOffsetStart: 0, SliceOffsetEnd: fileSize}, fileName); err != nil {
		utils.ErrorLog("failed uploading stream tmp file", err.Error())
		return
	}
	p, err := requests.RequestUploadFile(ctx, fileName, fileHash, fileSize, walletAddr, pubkey, signature, nil, false, true, desiredTier, allowHigherTier)
	if err != nil {
		utils.ErrorLog("failed creating RequestUploadFile", err.Error())
		return
	}
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, p, header.ReqUploadFile)
}

func (api *rpcPrivApi) RequestRegisterNewPP(ctx context.Context, param rpc_api.ParamReqRP) rpc_api.RPResult {
	metrics.RpcReqCount.WithLabelValues("RequestRegisterNewPP").Inc()
	reqId := uuid.New().String()
	ctx = core.RegisterRemoteReqId(ctx, reqId)
	event.RegisterNewPP(ctx)
	ctx, cancel := context.WithTimeout(ctx, INIT_WAIT_TIMEOUT)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			result := &rpc_api.RPResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found := pp.GetRPResult(setting.Config.P2PAddress + setting.WalletAddress + reqId)
			if result != nil && found {
				return *result
			}
		}
	}
}

func (api *rpcPrivApi) RequestActivate(ctx context.Context, param rpc_api.ParamReqActivate) rpc_api.ActivateResult {
	metrics.RpcReqCount.WithLabelValues("RequestActivate").Inc()
	deposit, err := utiltypes.ParseCoinNormalized(param.Deposit)
	if err != nil {
		return rpc_api.ActivateResult{Return: rpc_api.WRONG_INPUT}
	}
	fee, err := utiltypes.ParseCoinNormalized(param.Fee)
	if err != nil {
		return rpc_api.ActivateResult{Return: rpc_api.WRONG_INPUT}
	}

	txFee := utiltypes.TxFee{
		Fee:      fee,
		Gas:      param.Gas,
		Simulate: false,
	}
	reqId := uuid.New().String()
	ctx = core.RegisterRemoteReqId(ctx, reqId)
	err = event.Activate(ctx, deposit, txFee)
	if err != nil {
		return rpc_api.ActivateResult{Return: rpc_api.WRONG_INPUT}
	}

	//var done = make(chan bool)
	ctx, cancel := context.WithTimeout(ctx, INIT_WAIT_TIMEOUT)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			result := &rpc_api.ActivateResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found := pp.GetActivateResult(setting.WalletAddress + reqId)
			if result != nil && found {
				return *result
			}
		}
	}
}

func (api *rpcPrivApi) RequestPrepay(ctx context.Context, param rpc_api.ParamReqPrepay) rpc_api.PrepayResult {
	metrics.RpcReqCount.WithLabelValues("RequestPrepay").Inc()
	beneficiaryAddr, err := utiltypes.WalletAddressFromBech(setting.WalletAddress)
	if err != nil {
		return rpc_api.PrepayResult{Return: rpc_api.WRONG_WALLET_ADDRESS}
	}
	if len(param.WalletAddr) > 0 {
		beneficiaryAddr, err = utiltypes.WalletAddressFromBech(param.WalletAddr)
		if err != nil {
			return rpc_api.PrepayResult{Return: rpc_api.WRONG_WALLET_ADDRESS}
		}
	}

	prepayAmount, err := utiltypes.ParseCoinNormalized(param.PrepayAmount)
	if err != nil {
		return rpc_api.PrepayResult{Return: rpc_api.WRONG_INPUT}
	}
	fee, err := utiltypes.ParseCoinNormalized(param.Fee)
	if err != nil {
		return rpc_api.PrepayResult{Return: rpc_api.WRONG_INPUT}
	}

	txFee := utiltypes.TxFee{
		Fee:      fee,
		Gas:      param.Gas,
		Simulate: false,
	}
	reqId := uuid.New().String()
	ctx = core.RegisterRemoteReqId(ctx, reqId)
	err = event.Prepay(ctx, beneficiaryAddr.Bytes(), prepayAmount, txFee)
	if err != nil {
		return rpc_api.PrepayResult{Return: rpc_api.WRONG_INPUT}
	}

	//var done = make(chan bool)
	ctx, cancel := context.WithTimeout(ctx, INIT_WAIT_TIMEOUT)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			result := &rpc_api.PrepayResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found := pp.GetPrepayResult(setting.WalletAddress + reqId)
			if result != nil && found {
				return *result
			}
		}
	}
}

func (api *rpcPrivApi) RequestStartMining(ctx context.Context, param rpc_api.ParamReqStartMining) rpc_api.StartMiningResult {
	metrics.RpcReqCount.WithLabelValues("RequestStartMining").Inc()
	reqId := uuid.New().String()
	ctx = core.RegisterRemoteReqId(ctx, reqId)
	network.GetPeer(ctx).StartMining(ctx)

	ctx, cancel := context.WithTimeout(ctx, INIT_WAIT_TIMEOUT)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			result := &rpc_api.StartMiningResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found := pp.GetStartMiningResult(setting.Config.P2PAddress + reqId)
			if result != nil && found {
				return *result
			}
		}
	}
}
