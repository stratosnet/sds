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
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/crypto"
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
	"github.com/stratosnet/sds/pp/metrics"
	"github.com/stratosnet/sds/pp/namespace/stratoschain"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
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

	UPLOAD_SLICE_LOCAL_HANDLE_TIME = 60 * time.Second

	MAX_NUMBER_FILE_UPLOAD_AT_SAME_TIME = 20
)

var (
	uploadOffset = &sync.Map{}
	nfup         atomic.Int64
	wait         = make(chan bool)
)

type fileUploadOffset struct {
	PacketFileOffset uint64
	SliceFileOffset  uint64
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
		var f fileUploadOffset
		var e uint64
		// have to cut the requested data block into smaller pieces when the size is greater than the limit
		if end-start > FILE_DATA_SAFE_SIZE {
			f = fileUploadOffset{PacketFileOffset: start + FILE_DATA_SAFE_SIZE, SliceFileOffset: end}
			e = start + FILE_DATA_SAFE_SIZE
		} else {
			f = fileUploadOffset{PacketFileOffset: end, SliceFileOffset: end}
			e = end
		}
		uploadOffset.Store(fileHash, f)
		nr := &rpc_api.Result{
			Return:      r.Return,
			OffsetStart: &start,
			OffsetEnd:   &e,
		}
		return nr
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
	if !fwtypes.VerifyWalletSign(pubkey, signature, msgutils.GetFileUploadWalletSignMessage(fileHash, walletAddr, param.SequenceNumber, reqTime)) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}
	if _, ok := uploadOffset.Load(fileHash); ok {
		return rpc_api.Result{Return: rpc_api.CONFLICT_WITH_ANOTHER_SESSION}
	}

	nfup.Add(1)
	waitNumber := nfup.Load()
	if waitNumber >= MAX_NUMBER_FILE_UPLOAD_AT_SAME_TIME {
		<-wait
	}
	// fetch file slices from remote client and send upload request to sp
	fetchRemoteFileAndReqUpload := func() {
		metrics.UploadPerformanceLogNow(param.FileHash + ":RCV_REQ_UPLOAD_CLIENT")
		fileName := param.FileName
		fileSize := uint64(param.FileSize)
		sliceSize := uint64(setting.MaxSliceSize)
		sliceCount := uint64(math.Ceil(float64(fileSize) / float64(sliceSize)))
		defer func() {
			wait <- true
		}()
		defer func() {
			nfup.Add(-1)
		}()
		defer uploadOffset.Delete(fileHash)
		var slices []*protos.SliceHashAddr
		for sliceNumber := uint64(1); sliceNumber <= sliceCount; sliceNumber++ {
			sliceOffset := requests.GetSliceOffset(sliceNumber, sliceCount, sliceSize, fileSize)

			tmpSliceName := uuid.NewString()
			var rawData []byte
			var err error
			if file.CacheRemoteFileData(fileHash, sliceOffset, fileHash, tmpSliceName, false) != nil {
				return
			}

			rawData, err = file.GetSliceDataFromTmp(fileHash, tmpSliceName)
			if err != nil {
				_ = file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE, Detail: "failed reading slice data from cache" + err.Error()})
				return
			}

			sliceHash, err := crypto.CalcSliceHash(rawData, fileHash, sliceNumber)
			if err != nil {
				_ = file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE, Detail: "failed calculating slice hash" + err.Error()})
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
				_ = file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE, Detail: "failed renaming slice cache file" + err.Error()})
				return
			}
		}
		uploadOffset.Delete(fileHash)
		_ = file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.SUCCESS})

		var s rpc_api.Signature
		c, cancel := context.WithTimeout(context.Background(), INIT_WAIT_TIMEOUT)
		defer cancel()
		utils.DebugLog("LISTEN:", fileHash)
		select {
		case <-c.Done():
			return
		case sign := <-file.SubscribeFileUploadSign(fileHash):
			s = sign.Signature
			reqTime = sign.ReqTime
		}

		// start to upload file
		p, err := requests.RequestUploadFile(ctx, fileName, fileHash, fileSize, walletAddr, pubkey, s.Signature, reqTime,
			slices, false, param.DesiredTier, param.AllowHigherTier, 0)
		if err != nil {
			_ = file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE, Detail: "failed request upload file" + err.Error()})
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
	case result := <-fileEventCh:
		file.UnsubscribeRemoteFileEvent(fileHash)
		if result != nil {
			if result.Return == rpc_api.UPLOAD_DATA {
				f := fileUploadOffset{PacketFileOffset: *result.OffsetEnd, SliceFileOffset: *result.OffsetEnd}
				uploadOffset.Store(fileHash, f)
			}
			result = ResultHook(result, fileHash)
			return *result
		} else {
			result = &rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE, Detail: "failed result with no specific reason"}
			return *result
		}
	}
}

func (api *rpcPubApi) UploadData(ctx context.Context, param rpc_api.ParamUploadData) rpc_api.Result {
	metrics.UploadPerformanceLogNow(param.FileHash + ":RCV_REQ_UPLOAD_SP:")
	fileHash := param.FileHash
	walletAddr := param.Signature.Address
	pubkey := param.Signature.Pubkey
	signature := param.Signature.Signature
	reqTime := param.ReqTime
	stop := param.Stop

	// verify if wallet and public key match
	if !fwtypes.VerifyWalletAddr(pubkey, walletAddr) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}
	if !fwtypes.VerifyWalletSign(pubkey, signature, msgutils.GetFileUploadWalletSignMessage(fileHash, walletAddr, param.SequenceNumber, reqTime)) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	content := param.Data
	var dec []byte

	// first part: if the amount of bytes server requested haven't been finished,
	// go on asking from the client
	fuo, found := uploadOffset.Load(fileHash)
	if stop || !found {
		return rpc_api.Result{Return: rpc_api.SESSION_STOPPED}
	}
	fo := fuo.(fileUploadOffset)
	dec, _ = b64.StdEncoding.DecodeString(content)
	file.SendFileDataBack(fileHash, file.DataWithOffset{Data: dec, Offset: fo.PacketFileOffset})
	if fo.SliceFileOffset-fo.PacketFileOffset > FILE_DATA_SAFE_SIZE {
		start := fo.PacketFileOffset
		end := fo.PacketFileOffset + FILE_DATA_SAFE_SIZE
		nr := rpc_api.Result{
			Return:      rpc_api.UPLOAD_DATA,
			OffsetStart: &start,
			OffsetEnd:   &end,
		}

		fo.PacketFileOffset = fo.PacketFileOffset + FILE_DATA_SAFE_SIZE
		uploadOffset.Store(fileHash, fo)
		return nr
	}
	if fo.PacketFileOffset < fo.SliceFileOffset {
		start := fo.PacketFileOffset
		end := fo.SliceFileOffset
		nr := rpc_api.Result{
			Return:      rpc_api.UPLOAD_DATA,
			OffsetStart: &start,
			OffsetEnd:   &end,
		}
		fo.PacketFileOffset = fo.SliceFileOffset
		uploadOffset.Store(fileHash, fo)
		return nr
	}

	// second part: let the server decide what will be the next step
	newctx, cancel := context.WithTimeout(context.Background(), UPLOAD_SLICE_LOCAL_HANDLE_TIME)
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
			result = &rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE, Detail: "failed result with no specific reason"}
			return *result
		}
	}
}

func (api *rpcPubApi) UploadSign(ctx context.Context, param rpc_api.ParamUploadSign) rpc_api.Result {
	fileHash := param.FileHash
	walletAddr := param.Signature.Address
	pubkey := param.Signature.Pubkey
	signature := param.Signature.Signature
	reqTime := param.ReqTime

	// verify if wallet and public key match
	if !fwtypes.VerifyWalletAddr(pubkey, walletAddr) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}
	if !fwtypes.VerifyWalletSign(pubkey, signature, msgutils.GetFileUploadWalletSignMessage(fileHash, walletAddr, param.SequenceNumber, reqTime)) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	file.SetFileUploadSign(&param, fileHash)

	// second part: let the server decide what will be the next step
	newctx, cancel := context.WithTimeout(context.Background(), UPLOAD_SLICE_LOCAL_HANDLE_TIME)
	defer cancel()

	result := &rpc_api.Result{}
	select {
	case <-newctx.Done():
		result.Return = rpc_api.TIME_OUT
	// since a slice has been passed to the application, wait for application's reply then return the result back to the rpc client
	case result = <-file.SubscribeRemoteFileEvent(fileHash):
	}
	if result == nil {
		return rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE, Detail: "failed result with no specific reason"}
	}
	return *result
}

func (api *rpcPubApi) RequestUploadStream(ctx context.Context, param rpc_api.ParamReqUploadFile) rpc_api.Result {
	metrics.RpcReqCount.WithLabelValues("RequestUploadStream").Inc()
	fileHash := param.FileHash
	walletAddr := param.Signature.Address
	pubkey := param.Signature.Pubkey
	signature := param.Signature.Signature
	reqTime := param.ReqTime

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
				_ = file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE, Detail: "failed caching file data"})
				return
			}
		}

		tmpFilePath := filepath.Join(file.GetTmpFileFolderPath(file.TMP_FOLDER_VIDEO), fileName)
		defer os.RemoveAll(tmpFilePath) // remove tmp file no matter what the result is
		calculatedFileHash, err := crypto.CalcFileHash(tmpFilePath, "", crypto.VIDEO_CODEC)
		if err != nil {
			_ = file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE, Detail: "failed calculating slice hash" + err.Error()})
			return
		}

		if calculatedFileHash != fileHash {
			_ = file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.WRONG_FILE_INFO, Detail: "file hash doesn't match"})
			return
		}

		fileHandler := event.GetUploadFileHandler(true)
		fInfo, slices, err := fileHandler.PreUpload(ctx, tmpFilePath, "")
		if err != nil {
			_ = file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE, Detail: "failed handling pre_upload" + err.Error()})
			return
		}

		// start to upload file
		p, err := requests.RequestUploadFile(ctx, fileName, fileHash, fileSize, walletAddr, pubkey, signature, reqTime,
			slices, false, param.DesiredTier, param.AllowHigherTier, fInfo.Duration)
		if err != nil {
			_ = file.SetRemoteFileResult(fileHash, rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE, Detail: "failed request upload file" + err.Error()})
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
			result = &rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE, Detail: "failed result with no specific reason"}
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
	//var result *rpc_api.Result
	reqId := uuid.New().String()

	ctx = core.RegisterRemoteReqId(ctx, reqId)
	req := requests.ReqFileStorageInfoData(ctx, param.FileHandle, "", "", wallet, wpk.Bytes(), wsig, nil, param.ReqTime)
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, req, header.ReqFileStorageInfo)

	ctx, cancel := context.WithTimeout(ctx, INIT_WAIT_TIMEOUT)
	defer file.UnsubscribeDownloadSlice(fileHash + reqId)
	defer cancel()
	select {
	case <-ctx.Done():
		return rpc_api.Result{Return: rpc_api.TIME_OUT}
	case result := <-file.SubscribeDownloadSlice(fileHash + reqId):
		if result.Return != rpc_api.DOWNLOAD_OK {
			return *result
		}
		data, start, end, _ := file.NextRemoteDownloadPacket(fileHash, reqId)
		if data == nil {
			return rpc_api.Result{Return: rpc_api.FILE_REQ_FAILURE, Detail: "data is empty"}
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
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, req, header.ReqFileStorageInfo)

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
	data, start, end, finished := file.NextRemoteDownloadPacket(param.FileHash, param.ReqId)
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
			Detail: "data is empty",
		}
	}

	return *result
}

func (api *rpcPubApi) DownloadedFileInfo(ctx context.Context, param rpc_api.ParamDownloadFileInfo) rpc_api.Result {
	metrics.RpcReqCount.WithLabelValues("DownloadedFileInfo").Inc()
	return rpc_api.Result{Return: rpc_api.SUCCESS}
}

func (api *rpcPubApi) RequestDeleteFile(ctx context.Context, param rpc_api.ParamReqDeleteFile) rpc_api.Result {
	fileHash := param.FileHash
	walletAddr := param.Signature.Address
	pubkey := param.Signature.Pubkey
	signature := param.Signature.Signature
	reqTime := param.ReqTime

	// verify if wallet and public key match
	if !fwtypes.VerifyWalletAddr(pubkey, walletAddr) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	if !fwtypes.VerifyWalletSign(pubkey, signature, msgutils.DeleteShareWalletSignMessage(fileHash, walletAddr, reqTime)) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}
	pk, _ := fwtypes.WalletPubKeyFromBech32(pubkey)
	sigByte, _ := hex.DecodeString(signature)

	event.DeleteFile(ctx, fileHash, walletAddr, pk.Bytes(), sigByte, reqTime)

	// wait for the result
	ctx, cancel := context.WithTimeout(ctx, WAIT_TIMEOUT)
	defer cancel()

	defer file.UnsubscribeFileDeleteResult(fileHash)
	// wait for result, SUCCESS or some failure
	select {
	case <-ctx.Done():
		result := &rpc_api.Result{Return: rpc_api.TIME_OUT}
		return *result
	case result := <-file.SubscribeFileDeleteResult(fileHash):
		if result != nil {
			return *result
		} else {
			return rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE}
		}
	}
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
	// validate the length of meta info
	if len(param.MetaInfo) > fwtypes.MAX_META_INFO_LENGTH {
		return rpc_api.FileShareResult{Return: rpc_api.WRONG_INPUT + ", invalid meta info"}
	}
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
	event.ReqShareFile(reqCtx, param.FileHash, "", param.Signature.Address, param.Duration, param.PrivateFlag,
		wpk.Bytes(), wsig, param.ReqTime, param.IpfsCid, param.MetaInfo)

	// wait for result, SUCCESS or some failure
	var result *rpc_api.FileShareResult

	defer file.UnsubscribeFileShareResult(param.Signature.Address + reqId)
	select {
	case <-ctx.Done():
		result = &rpc_api.FileShareResult{Return: rpc_api.TIME_OUT}
		return *result
	case result = <-file.SubscribeFileShareResult(param.Signature.Address + reqId):
		if result != nil {
			return *result
		} else {
			return rpc_api.FileShareResult{Return: rpc_api.INTERNAL_DATA_FAILURE}
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

	defer file.UnsubscribeFileShareResult(param.Signature.Address + reqId)
	select {
	case <-ctx.Done():
		result = &rpc_api.FileShareResult{Return: rpc_api.TIME_OUT}
		return *result
	case result = <-file.SubscribeFileShareResult(param.Signature.Address + reqId):
		if result != nil {
			return *result
		} else {
			return rpc_api.FileShareResult{Return: rpc_api.INTERNAL_DATA_FAILURE}
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

	defer file.UnsubscribeFileShareResult(param.Signature.Address + reqId)
	select {
	case <-ctx.Done():
		result = &rpc_api.FileShareResult{Return: rpc_api.TIME_OUT}
		return *result
	case result = <-file.SubscribeFileShareResult(param.Signature.Address + reqId):
		if result != nil {
			return *result
		} else {
			return rpc_api.FileShareResult{Return: rpc_api.INTERNAL_DATA_FAILURE}
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

	shareLink, err := fwtypes.ParseShareLink(param.ShareLink)
	if err != nil {
		utils.ErrorLog("wrong share link")
		return rpc_api.Result{Return: rpc_api.WRONG_INPUT}
	}

	ctx, cancel := context.WithTimeout(ctx, INIT_WAIT_TIMEOUT)
	defer cancel()
	reqId := uuid.New().String()
	reqCtx := core.RegisterRemoteReqId(ctx, reqId)
	req := requests.ReqGetShareFileData(shareLink.Link, shareLink.Password, "", param.Signature.Address,
		p2pserver.GetP2pServer(reqCtx).GetP2PAddress().String(), wpk.Bytes(), wsig, param.ReqTime)
	p2pserver.GetP2pServer(reqCtx).SendMessageToSPServer(reqCtx, req, header.ReqGetShareFile)

	key := shareLink.Link + reqId
	defer file.UnsubscribeFileShareResult(key)
	select {
	case <-ctx.Done():
		return rpc_api.Result{Return: rpc_api.TIME_OUT}
	case result := <-file.SubscribeFileShareResult(key):
		if result == nil {
			return rpc_api.Result{Return: rpc_api.INTERNAL_DATA_FAILURE}
		}
		if result.Return != rpc_api.SUCCESS && result.Return != rpc_api.SHARED_DL_START {
			return rpc_api.Result{Return: result.Return, Detail: result.Detail}
		}
		fileHash := result.FileInfo[0].FileHash
		data, start, end, _ := file.NextRemoteDownloadPacket(fileHash, reqId)
		if data == nil {
			return rpc_api.Result{Return: rpc_api.FILE_REQ_FAILURE, Detail: "data is empty"}
		}

		re := rpc_api.Result{
			Return:      rpc_api.DOWNLOAD_OK,
			OffsetStart: &start,
			OffsetEnd:   &end,
			FileHash:    fileHash,
			FileName:    file.GetFileName(fileHash),
			FileData:    b64.StdEncoding.EncodeToString(data),
			ReqId:       reqId,
		}
		if len(result.FileInfo) != 0 {
			re.FileSize = result.FileInfo[0].FileSize
		}
		return re
	}
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

	shareLink, err := fwtypes.ParseShareLink(param.ShareLink)
	if err != nil {
		utils.ErrorLog("wrong share link")
		return rpc_api.Result{Return: rpc_api.WRONG_INPUT}
	}

	reqId := uuid.New().String()
	ctx, cancel := context.WithTimeout(ctx, WAIT_TIMEOUT)
	defer cancel()
	key := shareLink.Link + reqId

	reqCtx := core.RegisterRemoteReqId(ctx, reqId)
	req := requests.ReqGetShareFileData(shareLink.Link, shareLink.Password, "", param.Signature.Address,
		p2pserver.GetP2pServer(reqCtx).GetP2PAddress().String(), wpk.Bytes(), wsig, param.ReqTime)
	p2pserver.GetP2pServer(reqCtx).SendMessageToSPServer(reqCtx, req, header.ReqGetShareFile)

	// the application gives FileShareResult type of result
	var res *rpc_api.FileShareResult

	defer file.UnsubscribeFileShareResult(key)
	select {
	case <-ctx.Done():
		return rpc_api.Result{Return: rpc_api.TIME_OUT}
	case res = <-file.SubscribeFileShareResult(key):
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

func (api *rpcPrivApi) RequestUpdatePPInfo(ctx context.Context, param rpc_api.ParamReqUpdatePPInfo) rpc_api.UpdatePPInfoResult {
	metrics.RpcReqCount.WithLabelValues("RequestUpdatePPInfo").Inc()
	var err error
	_, err = fwtypes.WalletAddressFromBech32(setting.WalletAddress)
	if err != nil {
		return rpc_api.UpdatePPInfoResult{Return: rpc_api.WRONG_WALLET_ADDRESS}
	}
	fee, err := txclienttypes.ParseCoinNormalized(param.Fee)
	if err != nil {
		return rpc_api.UpdatePPInfoResult{Return: rpc_api.WRONG_INPUT}
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

	err = stratoschain.UpdateResourceNode(ctx, param.Moniker, param.Identity, param.Website, param.SecurityContact, param.Details, txFee)
	if err != nil {
		return rpc_api.UpdatePPInfoResult{Return: rpc_api.WRONG_INPUT}
	}

	for {
		select {
		case <-ctx.Done():
			result := &rpc_api.UpdatePPInfoResult{Return: rpc_api.TIME_OUT}
			return *result
		default:
			result, found := pp.GetUpdatePPInfoResult(setting.WalletAddress + reqId)
			if result != nil && found {
				return *result
			}
		}
	}
}
