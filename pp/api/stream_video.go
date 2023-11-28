package api

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"

	"github.com/stratosnet/sds/framework/types/bech32"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/framework/utils/crypto/ed25519"
	"github.com/stratosnet/sds/framework/utils/datamesh"
	"github.com/stratosnet/sds/framework/utils/httpserv"
	fwutiltypes "github.com/stratosnet/sds/framework/utils/types"
	rpc_api "github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/namespace"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/sds-msg/header"
	"github.com/stratosnet/sds/sds-msg/protos"
)

type StreamReqBody struct {
	FileHash      string
	FileName      string
	WalletAddress string
	P2PAddress    string
	SpP2pAddress  string
	RestAddress   string
	Sign          []byte
	SavePath      string
	FileReqId     string
	FileTimestamp int64
	SliceInfo     *protos.DownloadSliceInfo
	SliceInfos    []*protos.DownloadSliceInfo
}

type StreamInfo struct {
	HeaderFile         string
	SegmentToSliceInfo map[string]*protos.DownloadSliceInfo
	FileInfo           *protos.RspFileStorageInfo
}

type SliceInfo struct {
	SliceHash string
	TaskId    string
}

type SharedFileInfo struct {
	FileHash     string
	OwnerAddress string
}

func streamVideoInfoCache(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ownerWalletAddress, fileHash, err := parseFilePath(req.RequestURI)
	if err != nil {
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	if setting.State == types.PP_ACTIVE {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "Current node is activated and is not allowed to cache video").ToBytes())
		return
	}

	sn, err := handleGetOzone(ctx, setting.WalletAddress)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "failed to get ozone").ToBytes())
		return
	}

	sdmPath := datamesh.DataMeshId{
		Owner: ownerWalletAddress,
		Hash:  fileHash,
	}.String()

	r := reqDownloadMsg(fileHash, sdmPath, sn)
	res := namespace.RpcPubApi().RequestVideoDownload(ctx, r)

	if res.Return != rpc_api.DOWNLOAD_OK {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "failed to get file storage info").ToBytes())
		return
	}

	streamInfo, err := getStreamInfo(ctx, fileHash, res.ReqId)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	ret, _ := json.Marshal(streamInfo)
	_, _ = w.Write(ret)

	cacheVideoSlices(ctx, streamInfo.FileInfo)
}

func streamSharedVideoInfoCache(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	if setting.State == types.PP_ACTIVE {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "Current node is activated and is not allowed to cache video").ToBytes())
		return
	}

	sn, err := handleGetOzone(ctx, setting.WalletAddress)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "failed to get ozone").ToBytes())
		return
	}

	shareLink, _, _ := parseShareLink(req.RequestURI)
	reqGetSharedMsg := reqGetSharedMsg(shareLink)
	res := namespace.RpcPubApi().RequestGetShared(ctx, reqGetSharedMsg)

	if res.Return != rpc_api.SHARED_DL_START {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "failed to get file storage info").ToBytes())
		return
	}

	fileHash := res.FileHash
	reqDownloadShared := reqDownloadShared(fileHash, sn, res.ReqId)
	res = namespace.RpcPubApi().RequestDownloadSharedVideo(ctx, reqDownloadShared)
	if res.Return != rpc_api.DOWNLOAD_OK {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "failed to get file storage info").ToBytes())
		return
	}

	streamInfo, err := getStreamInfo(ctx, res.FileHash, res.ReqId)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	ret, _ := json.Marshal(streamInfo)
	_, _ = w.Write(ret)

	cacheVideoSlices(ctx, streamInfo.FileInfo)
}

func streamVideoInfoHttp(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ownerWalletAddress, fileHash, err := parseFilePath(req.RequestURI)
	if err != nil {
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	sn, err := handleGetOzone(ctx, ownerWalletAddress)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "failed to get ozone").ToBytes())
		return
	}

	sdmPath := datamesh.DataMeshId{
		Owner: ownerWalletAddress,
		Hash:  fileHash,
	}.String()

	r := reqDownloadMsg(fileHash, sdmPath, sn)
	res := namespace.RpcPubApi().RequestVideoDownload(ctx, r)

	if res.Return != rpc_api.DOWNLOAD_OK {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "failed to get file storage info").ToBytes())
		return
	}

	streamInfo, err := getStreamInfo(ctx, fileHash, res.ReqId)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	ret, _ := json.Marshal(streamInfo)
	_, _ = w.Write(ret)
}

func streamVideoP2P(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	sliceHash := parseSliceHash(req.URL)

	body, err := verifyStreamReqBody(req)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	if setting.State == types.PP_ACTIVE {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "Current node is activated and is not allowed to cache video").ToBytes())
		return
	}

	utils.DebugLog("Send request to retrieve the slice ", sliceHash)

	fInfo := &protos.RspFileStorageInfo{
		FileHash:      body.FileHash,
		SavePath:      body.SavePath,
		FileName:      body.FileName,
		NodeSign:      body.Sign,
		ReqId:         body.FileReqId,
		SpP2PAddress:  body.SpP2pAddress,
		WalletAddress: setting.WalletAddress,
		TimeStamp:     body.FileTimestamp,
		SliceInfo:     body.SliceInfos,
	}

	data, err := getSliceData(ctx, fInfo, body.SliceInfo)
	if err != nil {
		utils.ErrorLog("failed to get video slice ", err)
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "failed to get video slice").ToBytes())
		return
	}
	_, _ = w.Write(data)
}

func streamVideoHttp(w http.ResponseWriter, req *http.Request) {
	sliceHash := parseSliceHash(req.URL)

	body, err := verifyStreamReqBody(req)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	utils.DebugLog("Redirect the request to resource node.")
	redirectToResourceNode(body.FileHash, sliceHash, body.RestAddress, setting.WalletAddress, w, req)
	sendReportStreamResult(req.Context(), body, sliceHash, false)
}

func redirectToResourceNode(fileHash, sliceHash, restAddress, walletAddress string, w http.ResponseWriter, req *http.Request) {
	var targetAddress string
	if dlTask, ok := task.DownloadTaskMap.Load(fileHash + walletAddress + task.LOCAL_REQID); ok {
		//self is the resource PP and has task info
		downloadTask := dlTask.(*task.DownloadTask)
		if sliceInfo, ok := downloadTask.GetSliceInfo(sliceHash); ok {
			targetAddress = sliceInfo.StoragePpInfo.RestAddress
		} else {
			targetAddress = restAddress
		}
	} else {
		//to ask resource pp for slice addresses
		targetAddress = restAddress
	}
	redirectURL := fmt.Sprintf("http://%s/videoSlice/%s", targetAddress, sliceHash)
	utils.DebugLog("Redirect URL ", redirectURL)
	http.Redirect(w, req, redirectURL, http.StatusTemporaryRedirect)
}

func clearStreamTask(w http.ResponseWriter, req *http.Request) {
	event.ClearFileInfoAndDownloadTask(req.Context(), parseFileHash(req.URL), task.LOCAL_REQID, w)
}

func GetVideoSlice(w http.ResponseWriter, req *http.Request) {
	sliceHash := parseSliceHash(req.URL)

	utils.Log("Received get video slice request :", req.URL)

	body, err := verifyStreamReqBody(req)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	utils.DebugLog("The request body is ", body)

	if dlTask, ok := task.DownloadTaskMap.Load(body.FileHash + setting.WalletAddress + task.LOCAL_REQID); ok {
		utils.DebugLog("Found task info ", body)
		var ppInfo *protos.PPBaseInfo
		downloadTask := dlTask.(*task.DownloadTask)
		if sliceInfo, ok := downloadTask.GetSliceInfo(sliceHash); ok {
			ppInfo = sliceInfo.StoragePpInfo
		}

		if ppInfo.P2PAddress != p2pserver.GetP2pServer(req.Context()).GetP2PAddress() {
			utils.DebugLog("Current P2PAddress does not have the requested slice")
			targetAddress := ppInfo.RestAddress
			redirectURL := fmt.Sprintf("http://%s/videoSlice/%s", targetAddress, sliceHash)
			utils.DebugLog("Redirect the request to " + redirectURL)
			http.Redirect(w, req, redirectURL, http.StatusTemporaryRedirect)
			return
		}
	}

	utils.DebugLog("Start getting the slice from local storage", body)

	video, err := file.GetSliceData(sliceHash)
	if err != nil {
		utils.DebugLog("failed get slice data ", err.Error())
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "Could not find the video segment!").ToBytes())
		return
	}

	if !verifySignature(body, sliceHash, video) {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "Authorization failed!").ToBytes())
		return
	}

	utils.DebugLog("Found the slice and return", body)
	_, _ = w.Write(video)

	sendReportStreamResult(req.Context(), body, sliceHash, true)
}

func parseFilePath(reqURI string) (walletAddress, fileHash string, err error) {
	filePath := reqURI[strings.Index(reqURI[1:], "/")+2:]

	if filePath == "" || len(filePath) != 82 || filePath[41] != '/' {
		err = errors.New("invalid file path")
		return
	}

	walletAddress = filePath[0:41]
	fileHash = filePath[42:82]
	return
}

func parseShareLink(reqURI string) (shareLink, password string, err error) {
	u, err := url.Parse(reqURI)
	shareLink = u.Path[strings.Index(u.Path[1:], "/")+2:]
	params, _ := url.ParseQuery(u.RawQuery)
	password = ""
	if passwordParams, ok := params["password"]; ok {
		password = passwordParams[0]
	}
	return
}

func parseFileHash(reqURL *url.URL) string {
	reqPath := reqURL.Path
	return reqPath[strings.LastIndex(reqPath, "/")+1:]
}

func parseSliceHash(reqURL *url.URL) string {
	reqPath := reqURL.Path
	return reqPath[strings.LastIndex(reqPath, "/")+1:]
}

func cacheVideoSlices(ctx context.Context, fInfo *protos.RspFileStorageInfo) {
	slices := make([]*protos.DownloadSliceInfo, len(fInfo.SliceInfo))
	for i := 0; i < len(fInfo.SliceInfo); i++ {
		idx := uint64(len(fInfo.SliceInfo)) - fInfo.SliceInfo[i].SliceNumber
		slices[idx] = fInfo.SliceInfo[i]
	}

	cacheCh := make(chan bool, setting.StreamCacheMaxSlice)

	for i := 0; i < setting.StreamCacheMaxSlice; i++ {
		cacheCh <- true
	}

	for idx, sliceInfo := range slices {
		<-cacheCh
		go func(idx int, sliceInfo *protos.DownloadSliceInfo) {
			exist, _ := checkSliceExist(fInfo.FileHash, sliceInfo.SliceStorageInfo.SliceHash)
			if !exist {
				_, _ = getSliceData(ctx, fInfo, sliceInfo)
			}
			if idx < len(slices)-setting.StreamCacheMaxSlice {
				cacheCh <- true
			}
		}(idx, sliceInfo)
	}
	close(cacheCh)
}

func checkSliceExist(fileHash, sliceHash string) (bool, string) {
	folder := filepath.Join(file.GetTmpDownloadPath(), setting.VideoPath, fileHash)
	slicePath := filepath.Join(folder, sliceHash)
	return file.CheckFilePathEx(slicePath), slicePath
}

func getStreamInfo(ctx context.Context, fileHash, reqId string) (*StreamInfo, error) {
	var fInfo *protos.RspFileStorageInfo
	if f, ok := task.DownloadFileMap.Load(fileHash + reqId); ok {
		fInfo = f.(*protos.RspFileStorageInfo)
		utils.DebugLog("Received file storage info from sp ", fInfo)
	}

	if fInfo == nil {
		return nil, errors.New("http stream video failed to get file storage info!")
	}

	if !utils.IsVideoStream(fInfo.FileHash) {
		return nil, errors.New("the file was not uploaded as video stream")
	}

	hlsInfo, err := getHlsInfo(ctx, fInfo)
	if hlsInfo == nil || err != nil {
		return nil, errors.Wrap(err, "failed to get hls info!")
	}

	segmentToSliceInfo := make(map[string]*protos.DownloadSliceInfo, 0)
	for segment := range hlsInfo.SegmentToSlice {
		segmentInfo := getVideoSliceInfo(segment, fInfo, hlsInfo)
		segmentToSliceInfo[segment] = segmentInfo
	}
	StreamInfo := &StreamInfo{
		HeaderFile:         hlsInfo.HeaderFile,
		SegmentToSliceInfo: segmentToSliceInfo,
		FileInfo:           fInfo,
	}
	return StreamInfo, nil
}

func getHlsInfo(ctx context.Context, fInfo *protos.RspFileStorageInfo) (*file.HlsInfo, error) {
	sliceInfo := getSliceInfoBySliceNumber(fInfo, uint64(1))
	data, err := getSliceData(ctx, fInfo, sliceInfo)
	if err != nil {
		return nil, err
	}
	return file.LoadHlsInfoFromData(data)
}

func getSliceData(ctx context.Context, fInfo *protos.RspFileStorageInfo, sliceInfo *protos.DownloadSliceInfo) ([]byte, error) {
	exist, slicePath := checkSliceExist(fInfo.FileHash, sliceInfo.SliceStorageInfo.SliceHash)
	if exist {
		data, err := file.GetWholeFileData(slicePath)
		if err == nil {
			return data, nil
		}
	}

	r := reqDownloadDataMsg(fInfo, sliceInfo)
	res := namespace.RpcPubApi().RequestDownloadSliceData(ctx, r)

	if res.Return != rpc_api.DOWNLOAD_OK {
		return nil, errors.New("failed to get video slice")
	}

	decoded, err := base64.StdEncoding.DecodeString(res.FileData)
	if err != nil {
		return nil, err
	}
	fileMg, err := os.OpenFile(slicePath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		fileMg, err = file.CreateFolderAndReopenFile(filepath.Dir(slicePath), filepath.Base(slicePath))
		if err != nil {
			return nil, err
		}
	}
	defer func() {
		_ = fileMg.Close()
	}()
	_ = file.WriteFile(decoded, 0, fileMg)
	return decoded, nil
}

func verifyStreamReqBody(req *http.Request) (*StreamReqBody, error) {
	body, err := io.ReadAll(req.Body)
	defer req.Body.Close()

	if err != nil {
		return nil, err
	}

	var reqBody StreamReqBody
	err = json.Unmarshal(body, &reqBody)

	if err != nil {
		return nil, err
	}

	if _, err := cid.Decode(reqBody.FileHash); err != nil {
		return nil, errors.Wrap(err, "incorrect file fileHash")
	}

	if _, err := fwutiltypes.P2pAddressFromBech(reqBody.P2PAddress); err != nil {
		return nil, errors.Wrap(err, "incorrect P2P address")
	}

	if reqBody.FileName == "" {
		return nil, errors.New("please give file name")
	}

	if _, err := fwutiltypes.P2pAddressFromBech(reqBody.SpP2pAddress); err != nil {
		return nil, errors.Wrap(err, "incorrect SP P2P address")
	}

	return &reqBody, nil
}

func verifySignature(reqBody *StreamReqBody, sliceHash string, data []byte) bool {
	val, ok := setting.SPMap.Load(reqBody.SpP2pAddress)
	if !ok {
		utils.ErrorLog("cannot find sp info by given the SP address ", reqBody.SpP2pAddress)
		return false
	}

	spInfo, ok := val.(setting.SPBaseInfo)
	if !ok {
		utils.ErrorLog("Fail to parse SP info ", reqBody.SpP2pAddress)
		return false
	}

	_, pubKeyRaw, err := bech32.DecodeAndConvert(spInfo.P2PPublicKey)
	if err != nil {
		utils.ErrorLog("Error when trying to decode P2P pubKey bech32", err)
		return false
	}

	p2pPubKey := ed25519.PubKeyBytesToPubKey(pubKeyRaw)
	msg := []byte(reqBody.P2PAddress + reqBody.FileHash + header.ReqDownloadSlice.Name)
	if !p2pPubKey.VerifySignature(msg, reqBody.Sign) {
		return false
	}

	return sliceHash == utils.CalcSliceHash(data, reqBody.FileHash, reqBody.SliceInfo.SliceNumber)
}

func sendReportStreamResult(ctx context.Context, body *StreamReqBody, sliceHash string, isPP bool) {
	event.SendReportStreamingResult(ctx, &protos.RspDownloadSlice{
		SliceInfo:     &protos.SliceOffsetInfo{SliceHash: sliceHash},
		FileHash:      body.FileHash,
		WalletAddress: setting.WalletAddress,
		TaskId:        body.SliceInfo.TaskId,
	}, isPP)
}

func getVideoSliceInfo(sliceName string, fInfo *protos.RspFileStorageInfo, hlsInfo *file.HlsInfo) *protos.DownloadSliceInfo {
	sliceNumber := hlsInfo.SegmentToSlice[sliceName]
	sliceInfo := getSliceInfoBySliceNumber(fInfo, sliceNumber)
	return sliceInfo
}

func getSliceInfoBySliceNumber(fInfo *protos.RspFileStorageInfo, sliceNumber uint64) *protos.DownloadSliceInfo {
	for _, slice := range fInfo.SliceInfo {
		if slice.SliceNumber == sliceNumber {
			return slice
		}
	}
	return nil
}

func reqDownloadMsg(hash, sdmPath, sn string) rpc_api.ParamReqDownloadFile {
	// param
	nowSec := time.Now().Unix()
	sign, _ := fwutiltypes.BytesToAccPriveKey(setting.WalletPrivateKey).Sign([]byte(utils.GetFileDownloadWalletSignMessage(hash, setting.WalletAddress, sn, nowSec)))
	walletPublicKey, _ := fwutiltypes.BytesToAccPubKey(setting.WalletPublicKey).ToBech()
	walletSign := rpc_api.Signature{
		Address:   setting.WalletAddress,
		Pubkey:    walletPublicKey,
		Signature: hex.EncodeToString(sign),
	}
	return rpc_api.ParamReqDownloadFile{
		FileHandle: sdmPath,
		Signature:  walletSign,
		ReqTime:    nowSec,
	}
}

func reqDownloadDataMsg(fInfo *protos.RspFileStorageInfo, sliceInfo *protos.DownloadSliceInfo) rpc_api.ParamReqDownloadData {
	return rpc_api.ParamReqDownloadData{
		FileHash:       fInfo.FileHash,
		ReqId:          fInfo.ReqId,
		SliceHash:      sliceInfo.SliceStorageInfo.SliceHash,
		SliceNumber:    sliceInfo.SliceNumber,
		SliceSize:      sliceInfo.SliceStorageInfo.SliceSize,
		NetworkAddress: sliceInfo.StoragePpInfo.NetworkAddress,
		P2PAddress:     sliceInfo.StoragePpInfo.P2PAddress,
	}
}

func reqGetSharedMsg(shareLink string) rpc_api.ParamReqGetShared {
	nowSec := time.Now().Unix()
	sign, _ := fwutiltypes.BytesToAccPriveKey(setting.WalletPrivateKey).Sign([]byte(utils.GetShareFileWalletSignMessage(shareLink, setting.WalletAddress, nowSec)))
	walletPublicKey, _ := fwutiltypes.BytesToAccPubKey(setting.WalletPublicKey).ToBech()
	walletSign := rpc_api.Signature{
		Address:   setting.WalletAddress,
		Pubkey:    walletPublicKey,
		Signature: hex.EncodeToString(sign),
	}
	return rpc_api.ParamReqGetShared{
		Signature: walletSign,
		ReqTime:   nowSec,
		ShareLink: shareLink,
	}
}

func reqDownloadShared(fileHash, sn, reqId string) rpc_api.ParamReqDownloadShared {
	nowSec := time.Now().Unix()
	sign, _ := fwutiltypes.BytesToAccPriveKey(setting.WalletPrivateKey).Sign([]byte(utils.GetFileDownloadWalletSignMessage(fileHash, setting.WalletAddress, sn, nowSec)))
	walletPublicKey, _ := fwutiltypes.BytesToAccPubKey(setting.WalletPublicKey).ToBech()
	walletSign := rpc_api.Signature{
		Address:   setting.WalletAddress,
		Pubkey:    walletPublicKey,
		Signature: hex.EncodeToString(sign),
	}
	return rpc_api.ParamReqDownloadShared{
		FileHash:  fileHash,
		Signature: walletSign,
		ReqTime:   nowSec,
		ReqId:     reqId,
	}
}

func handleGetOzone(ctx context.Context, walletAddress string) (string, error) {
	utils.Log("- request ozone balance (method: user_requestGetOzone)")
	res := namespace.RpcPubApi().RequestGetOzone(ctx, rpc_api.ParamReqGetOzone{
		WalletAddr: walletAddress,
	})

	if res.Return == rpc_api.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
		ozone, _ := strconv.ParseFloat(res.Ozone, 64)
		utils.Log("OZONE balance: ", ozone/1000000000.0)
		utils.Log("SN:            ", res.SequenceNumber)
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}

	return res.SequenceNumber, nil
}
