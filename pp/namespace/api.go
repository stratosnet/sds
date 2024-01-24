package namespace

import (
	"context"
	b64 "encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/crypto"
	"github.com/stratosnet/sds/framework/metrics"
	"github.com/stratosnet/sds/framework/msg/header"
	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/sds-msg/protos"
	msgutils "github.com/stratosnet/sds/sds-msg/utils"
	txclienttypes "github.com/stratosnet/sds/tx-client/types"

	"github.com/stratosnet/sds/pp"
	rpc_api "github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/namespace/stratoschain"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	pptypes "github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/rpc"
)

const (
	// FILE_DATA_SAFE_SIZE the length of request shall be shorter than 5242880 bytes
	// this equals 3932160 bytes after
	FILE_DATA_SAFE_SIZE = 3500000

	// WAIT_TIMEOUT timeout for waiting result from external source, in seconds
	WAIT_TIMEOUT time.Duration = 10 * time.Second

	// INIT_WAIT_TIMEOUT timeout for waiting the initial request
	INIT_WAIT_TIMEOUT time.Duration = 15 * time.Second

	SIGNATURE_INFO_TTL = 10 * time.Minute
)

var (
	// key: fileHash value: file
	FileOffset      = make(map[string]*FileFetchOffset)
	FileOffsetMutex sync.Mutex

	signatureInfoMap = utils.NewAutoCleanMap(SIGNATURE_INFO_TTL)
)

type FileFetchOffset struct {
	RemoteRequested   uint64
	ResourceNodeAsked uint64
}

type signatureInfo struct {
	signature rpc_api.Signature
	reqTime   int64
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
	walletAddr := param.Signature.Address
	pubkey := param.Signature.Pubkey
	signature := param.Signature.Signature
	reqTime := param.ReqTime

	// verify if wallet and public key match
	if !fwtypes.VerifyWalletAddr(pubkey, walletAddr) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// Store initial signature info
	signatureInfoMap.Store(fileHash, signatureInfo{
		signature: param.Signature,
		reqTime:   reqTime,
	})

	// fetch file slices from remote client and send upload request to sp
	fetchRemoteFileAndReqUpload := func() {
		metrics.UploadPerformanceLogNow(param.FileHash + ":RCV_REQ_UPLOAD_CLIENT")
		fileName := param.FileName
		fileSize := uint64(param.FileSize)
		sliceSize := uint64(setting.MaxSliceSize)
		sliceCount := uint64(math.Ceil(float64(fileSize) / float64(sliceSize)))

		var slices []*protos.SliceHashAddr
		for sliceNumber := uint64(1); sliceNumber <= sliceCount; sliceNumber++ {
			sliceOffset := requests.GetSliceOffset(sliceNumber, sliceCount, sliceSize, fileSize)

			tmpSliceName := uuid.NewString()
			var rawData []byte
			var err error
			if file.CacheRemoteFileData(fileHash, sliceOffset, fileHash, tmpSliceName, false) == nil {
				rawData, err = file.GetSliceDataFromTmp(fileHash, tmpSliceName)
				if err != nil {
					file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE})
					return
				}
			}

			//Encrypt slice data if required
			sliceHash, err := crypto.CalcSliceHash(rawData, fileHash, sliceNumber)
			if err != nil {
				file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE})
				return
			}

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

		// Get latest signature info and reqTime
		if info, found := signatureInfoMap.Load(fileHash); found {
			sigInfo := info.(signatureInfo)
			signature = sigInfo.signature.Signature
			reqTime = sigInfo.reqTime
		}

		// start to upload file
		p, err := requests.RequestUploadFile(ctx, fileName, fileHash, fileSize, walletAddr, pubkey, signature, reqTime,
			slices, false, param.DesiredTier, param.AllowHigherTier, 0)
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

	if param.Signature.Signature != "" {
		// Update signature info every time new data is received
		signatureInfoMap.Store(param.FileHash, signatureInfo{
			signature: param.Signature,
			reqTime:   param.ReqTime,
		})
	}

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
	metrics.RpcReqCount.WithLabelValues("RequestUploadStream").Inc()
	fileHash := param.FileHash
	walletAddr := param.Signature.Address
	pubkey := param.Signature.Pubkey
	signature := param.Signature.Signature
	reqTime := param.ReqTime

	// Store initial signature info
	signatureInfoMap.Store(fileHash, signatureInfo{
		signature: param.Signature,
		reqTime:   reqTime,
	})

	// fetch file slices from remote client and send upload request to sp
	fetchRemoteFileAndReqUpload := func() {
		metrics.UploadPerformanceLogNow(param.FileHash + ":RCV_REQ_UPLOAD_CLIENT")
		fileName := param.FileName
		fileSize := uint64(param.FileSize)
		sliceSize := uint64(setting.MaxSliceSize)
		sliceCount := uint64(math.Ceil(float64(fileSize) / float64(sliceSize)))

		var slices []*protos.SliceHashAddr
		for sliceNumber := uint64(1); sliceNumber <= sliceCount; sliceNumber++ {
			sliceOffset := requests.GetSliceOffset(sliceNumber, sliceCount, sliceSize, fileSize)

			if file.CacheRemoteFileData(fileHash, sliceOffset, file.TMP_FOLDER_VIDEO, fileName, true) != nil {
				file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE})
				return
			}
		}

		tmpFilePath := filepath.Join(file.GetTmpFileFolderPath(file.TMP_FOLDER_VIDEO), fileName)
		defer os.RemoveAll(tmpFilePath) // remove tmp file no matter what the result is
		calculatedFileHash, err := crypto.CalcFileHash(tmpFilePath, "", crypto.VIDEO_CODEC)
		if err != nil {
			file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE})
			return
		}

		if calculatedFileHash != fileHash {
			file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.WRONG_FILE_INFO})
			return
		}

		fileHandler := event.GetUploadFileHandler(true)
		fInfo, slices, err := fileHandler.PreUpload(ctx, tmpFilePath, "")
		if err != nil {
			file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE})
			return
		}

		// Get latest signature info and reqTime
		if info, found := signatureInfoMap.Load(fileHash); found {
			sigInfo := info.(signatureInfo)
			signature = sigInfo.signature.Signature
			reqTime = sigInfo.reqTime
		}

		// start to upload file
		p, err := requests.RequestUploadFile(ctx, fileName, fileHash, fileSize, walletAddr, pubkey, signature, reqTime,
			slices, false, param.DesiredTier, param.AllowHigherTier, fInfo.Duration)
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

func (api *rpcPubApi) GetFileStatus(ctx context.Context, param rpc_api.ParamGetFileStatus) rpc_api.FileStatusResult {
	metrics.RpcReqCount.WithLabelValues("GetFileStatus").Inc()

	pubkey, err := fwtypes.P2PPubKeyFromBech32(param.Signature.Pubkey)
	if err != nil {
		return rpc_api.FileStatusResult{Return: rpc_api.WRONG_INPUT, Error: err.Error()}
	}
	signature, err := hex.DecodeString(param.Signature.Signature)
	if err != nil {
		return rpc_api.FileStatusResult{Return: rpc_api.WRONG_INPUT, Error: err.Error()}
	}

	reqId := uuid.New().String()
	ctx = core.RegisterRemoteReqId(ctx, reqId)

	if rsp := event.GetFileStatus(ctx, param.FileHash, param.Signature.Address, pubkey.Bytes(), signature, param.ReqTime); rsp != nil {
		// Result available already available. No need to wait
		return rpc_api.FileStatusResult{
			Return:          rpc_api.SUCCESS,
			Error:           rsp.Result.Msg,
			FileUploadState: rsp.State,
			UserHasFile:     rsp.UserHasFile,
			Replicas:        rsp.Replicas,
		}
	}

	key := param.FileHash + reqId

	// wait for the result
	ctx, cancel := context.WithTimeout(ctx, WAIT_TIMEOUT)
	defer cancel()

	var result *rpc_api.FileStatusResult

	select {
	case <-ctx.Done():
		result = &rpc_api.FileStatusResult{Return: rpc_api.TIME_OUT}
	case result = <-file.SubscribeGetFileStatusDone(key):
	}
	file.UnsubscribeGetFileStatusDone(key)

	return *result
}

func (api *rpcPubApi) RequestDownload(ctx context.Context, param rpc_api.ParamReqDownloadFile) rpc_api.Result {
	metrics.RpcReqCount.WithLabelValues("RequestDownload").Inc()
	_, ownerWalletAddress, fileHash, _, err := fwtypes.ParseFileHandle(param.FileHandle)
	if err != nil {
		return rpc_api.Result{Return: rpc_api.WRONG_INPUT}
	}
	wallet := param.Signature.Address
	pubkey := param.Signature.Pubkey
	signature := param.Signature.Signature

	if ownerWalletAddress != wallet {
		utils.ErrorLog("only the file owner is allowed to download via sdm url")
		return rpc_api.Result{Return: rpc_api.WRONG_WALLET_ADDRESS}
	}

	metrics.UploadPerformanceLogNow(fileHash + ":RCV_REQ_DOWNLOAD_CLIENT")

	// wallet pubkey and wallet signature will be carried in sds messages in []byte format
	wpk, err := fwtypes.WalletPubKeyFromBech32(pubkey)
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
	if !fwtypes.VerifyWalletAddrBytes(wpk.Bytes(), wallet) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// if the file is being downloaded in an existing download session
	var result *rpc_api.Result
	reqId := uuid.New().String()
	key := fileHash + reqId

	// if there is already downloading session in progress, wait for the result
	if task.CheckDownloadTask(fileHash, setting.WalletAddress, task.LOCAL_REQID) {
		success := <-task.SubscribeDownloadResult(key)
		task.UnsubscribeDownloadResult(key)
		if !success {
			return rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE}
		}
	}

	// check if the file is already cached in download folder and if it's valid
	err = file.CheckDownloadCache(fileHash)
	if err != nil {
		// rpc normal download initiate a local download
		ctx = core.RegisterRemoteReqId(ctx, task.LOCAL_REQID)
		req := requests.ReqFileStorageInfoData(ctx, param.FileHandle, "", "", wallet, wpk.Bytes(), wsig, nil, param.ReqTime)
		p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, req, header.ReqFileStorageInfo)
		success := <-task.SubscribeDownloadResult(key)
		task.UnsubscribeDownloadResult(key)
		if !success {
			return rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE}
		}
	}

	// check the cached file again before send it to the client
	err = file.CheckDownloadCache(fileHash)
	if err != nil {
		return rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE}
	}

	// initialize the cursor reading the file from the beginning
	data, start, end, _ := file.ReadDownloadCachedData(fileHash, reqId)
	if data == nil {
		return rpc_api.Result{Return: rpc_api.FILE_REQ_FAILURE}
	}
	result = &rpc_api.Result{
		Return:      rpc_api.DOWNLOAD_OK,
		OffsetStart: &start,
		OffsetEnd:   &end,
		FileName:    file.GetFileName(fileHash),
		FileData:    b64.StdEncoding.EncodeToString(data),
		ReqId:       reqId,
	}

	return *result
}

func (api *rpcPubApi) RequestVideoDownload(ctx context.Context, param rpc_api.ParamReqDownloadFile) rpc_api.Result {
	metrics.RpcReqCount.WithLabelValues("RequestDownload").Inc()
	_, _, fileHash, _, err := fwtypes.ParseFileHandle(param.FileHandle)
	if err != nil {
		return rpc_api.Result{Return: rpc_api.WRONG_INPUT}
	}

	metrics.UploadPerformanceLogNow(fileHash + ":RCV_REQ_DOWNLOAD_CLIENT")
	wallet := param.Signature.Address
	pubkey := param.Signature.Pubkey
	signature := param.Signature.Signature

	// wallet pubkey and wallet signature will be carried in sds messages in []byte format
	wpk, err := fwtypes.WalletPubKeyFromBech32(pubkey)
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
	if !fwtypes.VerifyWalletAddrBytes(wpk.Bytes(), wallet) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	reqId := uuid.New().String()
	ctx = core.RegisterRemoteReqId(ctx, reqId)
	// request for downloading file
	req := requests.RequestDownloadFile(ctx, fileHash, param.FileHandle, wallet, reqId, wsig, wpk.Bytes(), nil, param.ReqTime)
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
		P2PAddress:         p2pserver.GetP2pServer(ctx).GetP2PAddress().String(),
	}
	networkAddress := param.NetworkAddress
	msgKey := "download#" + param.FileHash + strconv.FormatUint(param.SliceNumber, 10) + param.P2PAddress + param.ReqId
	err := p2pserver.GetP2pServer(ctx).SendMessageByCachedConn(ctx, msgKey, networkAddress, req, header.ReqDownloadSlice, nil)
	if err != nil {
		return rpc_api.Result{Return: rpc_api.INTERNAL_COMM_FAILURE}
	}

	key := param.SliceHash + param.ReqId

	data := make([]byte, param.SliceSize)
	downloadedSize := uint64(0)
	for downloadedSize < param.SliceSize {
		select {
		case <-time.After(WAIT_TIMEOUT):
			return rpc_api.Result{Return: rpc_api.TIME_OUT}
		case result := <-file.SubscribeRemoteSliceEvent(key):
			file.UnsubscribeRemoteSliceEvent(key)
			start := *result.OffsetStart
			end := *result.OffsetEnd
			downloadedSize += end - start
			decoded, err := b64.StdEncoding.DecodeString(result.FileData)
			if err != nil {
				return rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE}
			}
			copy(data[start:], decoded)
			file.SetDownloadSliceDone(key)
		}
	}

	return rpc_api.Result{
		Return:   rpc_api.DOWNLOAD_OK,
		FileData: b64.StdEncoding.EncodeToString(data),
	}
}

func (api *rpcPubApi) DownloadData(ctx context.Context, param rpc_api.ParamDownloadData) rpc_api.Result {

	// download from the cached file
	var result *rpc_api.Result
	data, start, end, finished := file.ReadDownloadCachedData(param.FileHash, param.ReqId)
	if finished {
		result = &rpc_api.Result{
			Return: rpc_api.DL_OK_ASK_INFO,
		}
		return *result
	}
	if data != nil {
		result = &rpc_api.Result{
			Return:      rpc_api.DOWNLOAD_OK,
			OffsetStart: &start,
			OffsetEnd:   &end,
			FileName:    file.GetFileName(param.FileHash),
			FileData:    b64.StdEncoding.EncodeToString(data),
		}
	} else {
		result = &rpc_api.Result{
			Return: rpc_api.GENERIC_ERR,
		}
	}

	return *result
}

func (api *rpcPubApi) DownloadedFileInfo(ctx context.Context, param rpc_api.ParamDownloadFileInfo) rpc_api.Result {
	metrics.RpcReqCount.WithLabelValues("DownloadedFileInfo").Inc()
	return rpc_api.Result{Return: rpc_api.SUCCESS}
}

func (api *rpcPubApi) RequestList(ctx context.Context, param rpc_api.ParamReqFileList) rpc_api.FileListResult {
	metrics.RpcReqCount.WithLabelValues("RequestList").Inc()

	reqId := uuid.New().String()
	ctx, cancel := context.WithTimeout(ctx, WAIT_TIMEOUT)
	defer cancel()
	ctx = core.RegisterRemoteReqId(ctx, reqId)

	// convert wallet pubkey to []byte which format is to be used in protobuf messages
	wpk, err := fwtypes.WalletPubKeyFromBech32(param.Signature.Pubkey)
	if err != nil {
		result := &rpc_api.FileListResult{Return: rpc_api.SIGNATURE_FAILURE + ", wrong wallet pubkey"}
		return *result
	}
	// decode the hex encoded signature back to []byte which is used in protobuf messages
	wsig, err := hex.DecodeString(param.Signature.Signature)
	if err != nil {
		result := &rpc_api.FileListResult{Return: rpc_api.SIGNATURE_FAILURE + ", wrong wallet signature"}
		return *result
	}
	event.FindFileList(ctx, "", param.Signature.Address, param.PageId, "", 0, true,
		wpk.Bytes(), wsig, param.ReqTime)

	// wait for result, SUCCESS or some failure
	var result *rpc_api.FileListResult
	var found bool

	for {
		select {
		case <-ctx.Done():
			result = &rpc_api.FileListResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found = file.GetFileListResult(param.Signature.Address + reqId)
			if result != nil && found {
				return *result
			}
		}
	}
}

func (api *rpcPubApi) RequestClearExpiredShareLinks(ctx context.Context, param rpc_api.ParamReqClearExpiredShareLinks) rpc_api.ClearExpiredShareLinksResult {
	metrics.RpcReqCount.WithLabelValues("RequestClearExpiredShareLinks").Inc()

	reqId := uuid.New().String()
	ctx, cancel := context.WithTimeout(ctx, WAIT_TIMEOUT)
	defer cancel()
	ctx = core.RegisterRemoteReqId(ctx, reqId)
	// convert wallet pubkey to []byte which format is to be used in protobuf messages
	wpk, err := fwtypes.WalletPubKeyFromBech32(param.Signature.Pubkey)
	if err != nil {
		result := &rpc_api.ClearExpiredShareLinksResult{Return: rpc_api.SIGNATURE_FAILURE + ", wrong wallet pubkey"}
		return *result
	}
	// decode the hex encoded signature back to []byte which is used in protobuf messages
	wsig, err := hex.DecodeString(param.Signature.Signature)
	if err != nil {
		result := &rpc_api.ClearExpiredShareLinksResult{Return: rpc_api.SIGNATURE_FAILURE + ", wrong wallet signature"}
		return *result
	}
	event.ClearExpiredShareLinks(ctx, param.Signature.Address, wpk.Bytes(), wsig, param.ReqTime)

	// wait for result, SUCCESS or some failure
	var result *rpc_api.ClearExpiredShareLinksResult
	var found bool

	for {
		select {
		case <-ctx.Done():
			result = &rpc_api.ClearExpiredShareLinksResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found = file.GetClearExpiredShareLinksResult(param.Signature.Address + reqId)
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
	// convert wallet pubkey to []byte which format is to be used in protobuf messages
	wpk, err := fwtypes.WalletPubKeyFromBech32(param.Signature.Pubkey)
	if err != nil {
		result := &rpc_api.FileShareResult{Return: rpc_api.SIGNATURE_FAILURE + ", wrong wallet pubkey"}
		return *result
	}
	// decode the hex encoded signature back to []byte which is used in protobuf messages
	wsig, err := hex.DecodeString(param.Signature.Signature)
	if err != nil {
		result := &rpc_api.FileShareResult{Return: rpc_api.SIGNATURE_FAILURE + ", wrong wallet signature"}
		return *result
	}
	event.GetReqShareFile(reqCtx, param.FileHash, "", param.Signature.Address, param.Duration, param.PrivateFlag,
		wpk.Bytes(), wsig, param.ReqTime)

	// wait for result, SUCCESS or some failure
	var result *rpc_api.FileShareResult
	var found bool

	for {
		select {
		case <-ctx.Done():
			result = &rpc_api.FileShareResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found = file.GetFileShareResult(param.Signature.Address + reqId)
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

	// convert wallet pubkey to []byte which format is to be used in protobuf messages
	wpk, err := fwtypes.WalletPubKeyFromBech32(param.Signature.Pubkey)
	if err != nil {
		result := &rpc_api.FileShareResult{Return: rpc_api.SIGNATURE_FAILURE + ", wrong wallet pubkey"}
		return *result
	}
	// decode the hex encoded signature back to []byte which is used in protobuf messages
	wsig, err := hex.DecodeString(param.Signature.Signature)
	if err != nil {
		result := &rpc_api.FileShareResult{Return: rpc_api.SIGNATURE_FAILURE + ", wrong wallet signature"}
		return *result
	}
	event.GetAllShareLink(reqCtx, param.Signature.Address, param.PageId, wpk.Bytes(), wsig, param.ReqTime)

	// wait for result, SUCCESS or some failure
	var result *rpc_api.FileShareResult
	var found bool

	for {
		select {
		case <-ctx.Done():
			result = &rpc_api.FileShareResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found = file.GetFileShareResult(param.Signature.Address + reqId)
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

	// convert wallet pubkey to []byte which format is to be used in protobuf messages
	wpk, err := fwtypes.WalletPubKeyFromBech32(param.Signature.Pubkey)
	if err != nil {
		result := &rpc_api.FileShareResult{Return: rpc_api.SIGNATURE_FAILURE + ", wrong wallet pubkey"}
		return *result
	}
	// decode the hex encoded signature back to []byte which is used in protobuf messages
	wsig, err := hex.DecodeString(param.Signature.Signature)
	if err != nil {
		result := &rpc_api.FileShareResult{Return: rpc_api.SIGNATURE_FAILURE + ", wrong wallet signature"}
		return *result
	}
	event.DeleteShare(reqCtx, param.ShareId, param.Signature.Address, wpk.Bytes(), wsig, param.ReqTime)

	// wait for result, SUCCESS or some failure
	var result *rpc_api.FileShareResult
	var found bool

	for {
		select {
		case <-ctx.Done():
			result = &rpc_api.FileShareResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found = file.GetFileShareResult(param.Signature.Address + reqId)
			if result != nil && found {
				return *result
			}
		}
	}
}

func (api *rpcPubApi) RequestGetShared(ctx context.Context, param rpc_api.ParamReqGetShared) rpc_api.Result {
	metrics.RpcReqCount.WithLabelValues("RequestGetShared").Inc()
	wallet := param.Signature.Address
	pubkey := param.Signature.Pubkey

	// wallet pubkey and wallet signature will be carried in sds messages in []byte format
	wpk, err := fwtypes.WalletPubKeyFromBech32(pubkey)
	if err != nil {
		utils.ErrorLog("wrong wallet pubkey")
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// verify if wallet and public key match
	if !fwtypes.VerifyWalletAddrBytes(wpk.Bytes(), wallet) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// decode the hex encoded signature back to []byte which is used in protobuf messages
	wsig, err := hex.DecodeString(param.Signature.Signature)
	if err != nil {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	shareLink, err := pptypes.ParseShareLink(param.ShareLink)
	if err != nil {
		utils.ErrorLog("wrong share link")
		return rpc_api.Result{Return: rpc_api.WRONG_INPUT}
	}

	reqId := uuid.New().String()
	ctx, cancel := context.WithTimeout(ctx, WAIT_TIMEOUT)
	defer cancel()
	key := param.Signature.Address + reqId

	reqCtx := core.RegisterRemoteReqId(ctx, reqId)
	event.GetShareFile(reqCtx, shareLink.ShareLink, shareLink.Password, "", param.Signature.Address, wpk.Bytes(), wsig, param.ReqTime)

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
				if res == nil {
					return rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE}
				}

				fileHash := res.FileInfo[0].FileHash
				// if the file is being downloaded in an existing download session
				reqId := uuid.New().String()
				key := fileHash + reqId

				// if there is already downloading session in progress, wait for the result
				if task.CheckDownloadTask(fileHash, setting.WalletAddress, task.LOCAL_REQID) {
					success := <-task.SubscribeDownloadResult(key)
					task.UnsubscribeDownloadResult(key)
					if !success {
						return rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE}
					}
				}

				// check if the file is already cached in download folder and if it's valid
				err = file.CheckDownloadCache(fileHash)

				// check the cached file again before send it to the client
				err = file.CheckDownloadCache(fileHash)
				if err != nil {
					return rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE}
				}

				// initialize the cursor reading the file from the beginning
				data, start, end, _ := file.ReadDownloadCachedData(fileHash, reqId)
				if data == nil {
					return rpc_api.Result{Return: rpc_api.FILE_REQ_FAILURE}
				}

				// the result is read, but it's nil
				return rpc_api.Result{
					Return:      rpc_api.DOWNLOAD_OK,
					OffsetStart: &start,
					OffsetEnd:   &end,
					FileHash:    fileHash,
					FileName:    file.GetFileName(fileHash),
					FileData:    b64.StdEncoding.EncodeToString(data),
					ReqId:       reqId,
				}
			}
		}
	}
	return rpc_api.Result{Return: rpc_api.TIME_OUT}
}

func (api *rpcPubApi) RequestGetVideoShared(ctx context.Context, param rpc_api.ParamReqGetShared) rpc_api.Result {
	metrics.RpcReqCount.WithLabelValues("RequestGetShared").Inc()
	wallet := param.Signature.Address
	pubkey := param.Signature.Pubkey

	// wallet pubkey and wallet signature will be carried in sds messages in []byte format
	wpk, err := fwtypes.WalletPubKeyFromBech32(pubkey)
	if err != nil {
		utils.ErrorLog("wrong wallet pubkey")
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// verify if wallet and public key match
	if !fwtypes.VerifyWalletAddrBytes(wpk.Bytes(), wallet) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// decode the hex encoded signature back to []byte which is used in protobuf messages
	wsig, err := hex.DecodeString(param.Signature.Signature)
	if err != nil {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	shareLink, err := pptypes.ParseShareLink(param.ShareLink)
	if err != nil {
		utils.ErrorLog("wrong share link")
		return rpc_api.Result{Return: rpc_api.WRONG_INPUT}
	}

	reqId := uuid.New().String()
	ctx, cancel := context.WithTimeout(ctx, WAIT_TIMEOUT)
	defer cancel()
	key := param.Signature.Address + reqId

	reqCtx := core.RegisterRemoteReqId(ctx, reqId)
	event.GetShareFile(reqCtx, shareLink.ShareLink, shareLink.Password, "", param.Signature.Address, wpk.Bytes(), wsig, param.ReqTime)

	// the application gives FileShareResult type of result
	var res *rpc_api.FileShareResult

	found := false
	for !found {
		select {
		case <-ctx.Done():
			return rpc_api.Result{Return: rpc_api.TIME_OUT}
		default:
			res, found = file.GetFileShareResult(key)
			if found {
				if res == nil {
					return rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE}
				}

				fileHash := res.FileInfo[0].FileHash
				file.SaveRemoteFileHash(fileHash+reqId, "", 0)
				// if the file is being downloaded in an existing download session
				// the result is read, but it's nil
				return rpc_api.Result{
					Return:   rpc_api.DOWNLOAD_OK,
					FileHash: fileHash,
					ReqId:    reqId,
				}
			}
		}
	}
	return rpc_api.Result{Return: rpc_api.TIME_OUT}
}

func (api *rpcPubApi) RequestDownloadShared(ctx context.Context, param rpc_api.ParamReqDownloadShared) rpc_api.Result {
	// wallet pubkey and wallet signature will be carried in sds messages in []byte format
	wpk, err := fwtypes.WalletPubKeyFromBech32(param.Signature.Pubkey)
	if err != nil {
		utils.ErrorLog("wrong wallet pubkey")
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// verify if wallet and public key match
	if !fwtypes.VerifyWalletAddrBytes(wpk.Bytes(), param.Signature.Address) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	wsig, err := hex.DecodeString(param.Signature.Signature)
	if err != nil {
		utils.ErrorLog("wrong signature")
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// file hash should be given in the result message
	fileHash := param.FileHash
	if fileHash == "" {
		return rpc_api.Result{Return: rpc_api.WRONG_FILE_INFO}
	}

	file.SetSignature(param.FileHash+param.Signature.Address+param.ReqId, wsig)

	// if the file is being downloaded in an existing download session
	var result *rpc_api.Result
	reqId := uuid.New().String()
	key := fileHash + param.ReqId

	// if there is already downloading session in progress, wait for the result
	if task.CheckDownloadTask(fileHash, setting.WalletAddress, task.LOCAL_REQID) {
		success := <-task.SubscribeDownloadResult(key)
		task.UnsubscribeDownloadResult(key)
		if !success {
			return rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE}
		}
	}

	// check if the file is already cached in download folder and if it's valid
	err = file.CheckDownloadCache(fileHash)
	if err != nil {
		// rpc normal download initiate a local download
		ctx = core.RegisterRemoteReqId(ctx, task.LOCAL_REQID)
		filePath := event.GetFilePath(key)
		if event.GetFilePath(key) == "" {
			return rpc_api.Result{Return: rpc_api.INTERNAL_COMM_FAILURE}
		}
		req := requests.ReqFileStorageInfoData(ctx, filePath, "", "", param.Signature.Address, wpk.Bytes(), wsig, nil, param.ReqTime)
		p2pserver.GetP2pServer(ctx).SendMessageDirectToSPOrViaPP(ctx, req, header.ReqFileStorageInfo)
		success := <-task.SubscribeDownloadResult(key)
		task.UnsubscribeDownloadResult(key)
		if !success {
			return rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE}
		}
	}

	// check the cached file again before send it to the client
	err = file.CheckDownloadCache(fileHash)
	if err != nil {
		return rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE}
	}

	// initialize the cursor reading the file from the beginning
	data, start, end, _ := file.ReadDownloadCachedData(fileHash, reqId)
	if data == nil {
		return rpc_api.Result{Return: rpc_api.FILE_REQ_FAILURE}
	}
	result = &rpc_api.Result{
		Return:      rpc_api.DOWNLOAD_OK,
		OffsetStart: &start,
		OffsetEnd:   &end,
		FileName:    file.GetFileName(fileHash),
		FileData:    b64.StdEncoding.EncodeToString(data),
		ReqId:       reqId,
	}

	return *result
}

func (api *rpcPubApi) RequestDownloadSharedVideo(ctx context.Context, param rpc_api.ParamReqDownloadShared) rpc_api.Result {
	// wallet pubkey and wallet signature will be carried in sds messages in []byte format
	wpk, err := fwtypes.WalletPubKeyFromBech32(param.Signature.Pubkey)
	if err != nil {
		utils.ErrorLog("wrong wallet pubkey")
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// verify if wallet and public key match
	if !fwtypes.VerifyWalletAddrBytes(wpk.Bytes(), param.Signature.Address) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	wsig, err := hex.DecodeString(param.Signature.Signature)
	if err != nil {
		utils.ErrorLog("wrong signature")
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// file hash should be given in the result message
	fileHash := param.FileHash
	if fileHash == "" {
		return rpc_api.Result{Return: rpc_api.WRONG_FILE_INFO}
	}

	file.SetSignature(param.FileHash+param.Signature.Address+param.ReqId, wsig)

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

func (api *rpcPrivApi) RequestRegisterNewPP(ctx context.Context, param rpc_api.ParamReqRP) rpc_api.RPResult {
	metrics.RpcReqCount.WithLabelValues("RequestRegisterNewPP").Inc()
	reqId := uuid.New().String()
	ctx = core.RegisterRemoteReqId(ctx, reqId)
	nowSec := time.Now().Unix()
	// sign the wallet signature by wallet private key
	wsignMsg := msgutils.RegisterNewPPWalletSignMessage(setting.WalletAddress, nowSec)
	wsign, err := setting.WalletPrivateKey.Sign([]byte(wsignMsg))
	if err != nil {
		result := &rpc_api.RPResult{Return: rpc_api.SIGNATURE_FAILURE + ", wrong wallet signature"}
		return *result
	}
	event.RegisterNewPP(ctx, setting.WalletAddress, setting.WalletPublicKey.Bytes(), wsign, nowSec)
	ctx, cancel := context.WithTimeout(ctx, INIT_WAIT_TIMEOUT)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			result := &rpc_api.RPResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found := pp.GetRPResult(setting.Config.Keys.P2PAddress + setting.WalletAddress + reqId)
			if result != nil && found {
				return *result
			}
		}
	}
}

func (api *rpcPrivApi) RequestActivate(ctx context.Context, param rpc_api.ParamReqActivate) rpc_api.ActivateResult {
	metrics.RpcReqCount.WithLabelValues("RequestActivate").Inc()
	deposit, err := txclienttypes.ParseCoinNormalized(param.Deposit)
	if err != nil {
		return rpc_api.ActivateResult{Return: rpc_api.WRONG_INPUT}
	}
	fee, err := txclienttypes.ParseCoinNormalized(param.Fee)
	if err != nil {
		return rpc_api.ActivateResult{Return: rpc_api.WRONG_INPUT}
	}

	txFee := txclienttypes.TxFee{
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
	beneficiaryAddr, err := fwtypes.WalletAddressFromBech32(setting.WalletAddress)
	if err != nil {
		return rpc_api.PrepayResult{Return: rpc_api.WRONG_WALLET_ADDRESS}
	}
	if len(param.Signature.Address) > 0 {
		beneficiaryAddr, err = fwtypes.WalletAddressFromBech32(param.Signature.Address)
		if err != nil {
			return rpc_api.PrepayResult{Return: rpc_api.WRONG_WALLET_ADDRESS}
		}
	}

	prepayAmount, err := txclienttypes.ParseCoinNormalized(param.PrepayAmount)
	if err != nil {
		return rpc_api.PrepayResult{Return: rpc_api.WRONG_INPUT}
	}
	fee, err := txclienttypes.ParseCoinNormalized(param.Fee)
	if err != nil {
		return rpc_api.PrepayResult{Return: rpc_api.WRONG_INPUT}
	}

	txFee := txclienttypes.TxFee{
		Fee:      fee,
		Gas:      param.Gas,
		Simulate: false,
	}
	reqId := uuid.New().String()
	ctx = core.RegisterRemoteReqId(ctx, reqId)

	// convert wallet pubkey to []byte which format is to be used in protobuf messages
	wpk, err := fwtypes.WalletPubKeyFromBech32(param.Signature.Pubkey)
	if err != nil {
		result := &rpc_api.PrepayResult{Return: rpc_api.SIGNATURE_FAILURE + ", wrong wallet pubkey"}
		return *result
	}
	// decode the hex encoded signature back to []byte which is used in protobuf messages
	wsig, err := hex.DecodeString(param.Signature.Signature)
	if err != nil {
		result := &rpc_api.PrepayResult{Return: rpc_api.SIGNATURE_FAILURE + ", wrong wallet signature"}
		return *result
	}
	err = event.Prepay(ctx, beneficiaryAddr.Bytes(), prepayAmount, txFee, param.Signature.Address, wpk.Bytes(), wsig, param.ReqTime)
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
			result, found := pp.GetStartMiningResult(setting.Config.Keys.P2PAddress + reqId)
			if result != nil && found {
				return *result
			}
		}
	}
}

func (api *rpcPrivApi) RequestWithdraw(ctx context.Context, param rpc_api.ParamReqWithdraw) rpc_api.WithdrawResult {
	metrics.RpcReqCount.WithLabelValues("RequestWithdraw").Inc()
	amount, err := txclienttypes.ParseCoinNormalized(param.Amount)
	if err != nil {
		return rpc_api.WithdrawResult{Return: rpc_api.WRONG_INPUT}
	}
	_, err = fwtypes.WalletAddressFromBech32(setting.WalletAddress)
	if err != nil {
		return rpc_api.WithdrawResult{Return: rpc_api.WRONG_WALLET_ADDRESS}
	}
	targetAddr, err := fwtypes.WalletAddressFromBech32(param.TargetAddress)
	if err != nil {
		return rpc_api.WithdrawResult{Return: rpc_api.WRONG_WALLET_ADDRESS}
	}
	fee, err := txclienttypes.ParseCoinNormalized(param.Fee)
	if err != nil {
		return rpc_api.WithdrawResult{Return: rpc_api.WRONG_INPUT}
	}
	txFee := txclienttypes.TxFee{
		Fee:      fee,
		Simulate: true,
	}
	if param.Gas > 0 {
		txFee.Gas = param.Gas
		txFee.Simulate = false
	}

	reqId := uuid.New().String()
	ctx = core.RegisterRemoteReqId(ctx, reqId)
	ctx, cancel := context.WithTimeout(ctx, INIT_WAIT_TIMEOUT)
	defer cancel()

	err = stratoschain.Withdraw(ctx, amount, targetAddr, txFee)
	if err != nil {
		return rpc_api.WithdrawResult{Return: rpc_api.WRONG_INPUT}
	}

	for {
		select {
		case <-ctx.Done():
			result := &rpc_api.WithdrawResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found := pp.GetWithdrawResult(setting.WalletAddress + reqId)
			if result != nil && found {
				return *result
			}
		}
	}
}

func (api *rpcPrivApi) RequestSend(ctx context.Context, param rpc_api.ParamReqSend) rpc_api.SendResult {
	metrics.RpcReqCount.WithLabelValues("RequestSend").Inc()
	amount, err := txclienttypes.ParseCoinNormalized(param.Amount)
	if err != nil {
		return rpc_api.SendResult{Return: rpc_api.WRONG_INPUT}
	}
	_, err = fwtypes.WalletAddressFromBech32(setting.WalletAddress)
	if err != nil {
		return rpc_api.SendResult{Return: rpc_api.WRONG_WALLET_ADDRESS}
	}
	toAddr, err := fwtypes.WalletAddressFromBech32(param.To)
	if err != nil {
		return rpc_api.SendResult{Return: rpc_api.WRONG_WALLET_ADDRESS}
	}
	fee, err := txclienttypes.ParseCoinNormalized(param.Fee)
	if err != nil {
		return rpc_api.SendResult{Return: rpc_api.WRONG_INPUT}
	}
	txFee := txclienttypes.TxFee{
		Fee:      fee,
		Simulate: true,
	}
	if param.Gas > 0 {
		txFee.Gas = param.Gas
		txFee.Simulate = false
	}

	reqId := uuid.New().String()
	ctx = core.RegisterRemoteReqId(ctx, reqId)
	ctx, cancel := context.WithTimeout(ctx, INIT_WAIT_TIMEOUT)
	defer cancel()

	err = stratoschain.Send(ctx, amount, toAddr, txFee)
	if err != nil {
		return rpc_api.SendResult{Return: rpc_api.WRONG_INPUT}
	}

	for {
		select {
		case <-ctx.Done():
			result := &rpc_api.SendResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found := pp.GetSendResult(setting.WalletAddress + reqId)
			if result != nil && found {
				return *result
			}
		}
	}
}

func (api *rpcPrivApi) RequestStatus(ctx context.Context, param rpc_api.ParamReqStatus) rpc_api.StatusResult {
	metrics.RpcReqCount.WithLabelValues("RequestStatus").Inc()
	reqId := uuid.New().String()
	ctx = core.RegisterRemoteReqId(ctx, reqId)
	network.GetPeer(ctx).GetPPStatusFromSP(ctx)

	ctx, cancel := context.WithTimeout(ctx, INIT_WAIT_TIMEOUT)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			result := &rpc_api.StatusResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found := pp.GetStatusResult(setting.Config.Keys.P2PAddress + reqId)
			if result != nil && found {
				return *result
			}
		}
	}
}

func (api *rpcPubApi) RequestServiceStatus(ctx context.Context, param rpc_api.ParamReqServiceStatus) rpc_api.ServiceStatusResult {
	metrics.RpcReqCount.WithLabelValues("RequestServiceStatus").Inc()
	reqId := uuid.New().String()
	ctx = core.RegisterRemoteReqId(ctx, reqId)
	rpcResult := &rpc_api.ServiceStatusResult{Return: rpc_api.SUCCESS}

	regStatStr, onlineStatStr := "", ""
	regStat := network.GetPeer(ctx).GetStateFromFsm()
	switch regStat.Id {
	case network.STATE_NOT_REGISTERED:
		regStatStr = "Not registered"
		onlineStatStr = "OFFLINE"
	case network.STATE_REGISTERING:
		regStatStr = "Registering"
		onlineStatStr = "OFFLINE"
	case network.STATE_REGISTERED:
		regStatStr = "Registered"
		onlineStatStr = "ONLINE"
	default:
		regStatStr = "Unknown"
		onlineStatStr = "Unknown"
	}
	msgStr := fmt.Sprintf("Registration Status: %v | Mining: %v ", regStatStr, onlineStatStr)
	rpcResult.Message = msgStr
	return *rpcResult
}
