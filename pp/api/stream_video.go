package api

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	msgtypes "github.com/stratosnet/sds/sds-msg/types"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/stratosnet/sds/framework/crypto"
	"github.com/stratosnet/sds/framework/msg/header"
	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/framework/utils/httpserv"
	"github.com/stratosnet/sds/sds-msg/protos"
	msgutils "github.com/stratosnet/sds/sds-msg/utils"

	rpc_api "github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/namespace"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	pptypes "github.com/stratosnet/sds/pp/types"
)

const (
	streamInfoFile = "streamInfo"
)

var (
	//TODO to be replaced by other map implementation that has similar feature
	RequestInfoMap = utils.NewAutoCleanMap(1 * time.Hour)
)

type StreamInfoRequest struct {
	PubKey        string `json:"pubKey"`
	WalletAddress string `json:"walletAddress"`
	Signature     string `json:"signature"`
	ReqTime       int64  `json:"reqTime"`
}

type StreamInfoResponse struct {
	HeaderFile string `json:"headerFile"`
	ReqId      string `json:"reqId"`
}

type StreamInfo struct {
	HeaderFile         string                               `json:"header_file"`
	FileHash           string                               `json:"file_hash"`
	SegmentToSliceInfo map[string]*protos.DownloadSliceInfo `json:"segment_to_slice_info"`
}

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
}

type OzoneInfo struct {
	WalletAddress  string `json:"walletAddress"`
	SequenceNumber string `json:"sequenceNumber"`
}

func GetOzone(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	walletAddress := parseGetOZoneWalletAddress(req.URL)
	sn, err := handleGetOzone(ctx, walletAddress)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "failed to get ozone").ToBytes())
		return
	}
	ret, _ := json.Marshal(OzoneInfo{
		WalletAddress:  walletAddress,
		SequenceNumber: sn,
	})
	_, _ = w.Write(ret)
}

func PrepareVideoFileCache(w http.ResponseWriter, req *http.Request) {
	streamVideoInfoCacheHelper(w, req, getWalletSignFromRequest)
}

func streamVideoInfoCache(w http.ResponseWriter, req *http.Request) {
	streamVideoInfoCacheHelper(w, req, getWalletSignFromLocal)
}

func PrepareSharedVideoFileCache(w http.ResponseWriter, req *http.Request) {
	streamSharedVideoInfoCacheHelper(w, req, getWalletSignFromRequest)
}

func streamSharedVideoInfoCache(w http.ResponseWriter, req *http.Request) {
	streamSharedVideoInfoCacheHelper(w, req, getWalletSignFromLocal)
}

func streamVideoInfoCacheHelper(w http.ResponseWriter, req *http.Request, getSignature func(req *http.Request, fileHash string) (*rpc_api.Signature, int64, error)) {
	ctx := req.Context()

	if setting.State == msgtypes.PP_ACTIVE {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "Current node is activated and is not allowed to cache video").ToBytes())
		return
	}

	ownerWalletAddress, fileHash, err := parseFilePath(req.RequestURI)
	if err != nil {
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	walletSign, reqTime, err := getSignature(req, fileHash)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	if cached, streamInfo := checkVideoCached(fileHash, walletSign.Address); cached {
		reqId := uuid.New().String()
		RequestInfoMap.Store(reqId, streamInfo)
		respondStreamInfoRequest(w, streamInfo.HeaderFile, reqId)
		return
	}

	sdmPath := fwtypes.DataMeshId{
		Owner: ownerWalletAddress,
		Hash:  fileHash,
	}.String()

	r := reqDownloadMsg(sdmPath, walletSign, reqTime)
	res := namespace.RpcPubApi().RequestVideoDownload(ctx, r)

	if res.Return != rpc_api.DOWNLOAD_OK {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "failed to get file storage info").ToBytes())
		return
	}

	reqId := res.ReqId
	streamInfo, _, err := getStreamInfo(ctx, fileHash, reqId)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	RequestInfoMap.Store(reqId, streamInfo)
	_ = cacheStreamInfo(fileHash, walletSign.Address, streamInfo)

	respondStreamInfoRequest(w, streamInfo.HeaderFile, reqId)

	twoSlicesReadyCh := make(chan bool)
	go cacheVideoSlices(ctx, streamInfo, reqId, twoSlicesReadyCh)
	<-twoSlicesReadyCh
	close(twoSlicesReadyCh)
}

func streamSharedVideoInfoCacheHelper(w http.ResponseWriter, req *http.Request, getSignature func(req *http.Request, shareLink string) (*rpc_api.Signature, int64, error)) {
	ctx := req.Context()

	if setting.State == msgtypes.PP_ACTIVE {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "Current node is activated and is not allowed to cache video").ToBytes())
		return
	}

	shareLink, password, _ := parseShareLink(req.RequestURI)

	walletSign, reqTime, err := getSignature(req, shareLink)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	if cached, streamInfo := checkVideoCached(shareLink, walletSign.Address); cached {
		reqId := uuid.New().String()
		RequestInfoMap.Store(reqId, streamInfo)
		respondStreamInfoRequest(w, streamInfo.HeaderFile, reqId)
		return
	}

	reqGetSharedMsg := reqGetSharedMsg(pptypes.GetShareFile{ShareLink: shareLink, Password: password}, walletSign, reqTime)
	res := namespace.RpcPubApi().RequestGetVideoShared(ctx, reqGetSharedMsg)

	if res.Return != rpc_api.DOWNLOAD_OK {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "failed to get file storage info").ToBytes())
		return
	}

	reqId := res.ReqId
	streamInfo, _, err := getStreamInfo(ctx, res.FileHash, reqId)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	_ = cacheStreamInfo(shareLink, walletSign.Address, streamInfo)
	RequestInfoMap.Store(reqId, streamInfo)

	respondStreamInfoRequest(w, streamInfo.HeaderFile, reqId)

	twoSlicesReadyCh := make(chan bool)
	go cacheVideoSlices(ctx, streamInfo, reqId, twoSlicesReadyCh)
	<-twoSlicesReadyCh
	close(twoSlicesReadyCh)
}

func respondStreamInfoRequest(w http.ResponseWriter, headerFile, reqId string) {
	resp := StreamInfoResponse{
		HeaderFile: headerFile,
		ReqId:      reqId,
	}
	ret, _ := json.Marshal(resp)
	_, _ = w.Write(ret)
}

func checkVideoCached(fileLink, walletAddress string) (bool, *StreamInfo) {
	exists, fileInfoPath := checkStreamInfoExist(fileLink, walletAddress)
	if !exists {
		return false, nil
	}
	streamInfoRaw, err := file.GetWholeFileData(fileInfoPath)
	if err != nil {
		return false, nil
	}
	streamInfo := &StreamInfo{}
	if err = json.Unmarshal(streamInfoRaw, streamInfo); err != nil {
		return false, nil
	}
	for _, slice := range streamInfo.SegmentToSliceInfo {
		if slice.SliceStorageInfo.SliceHash == "" {
			return false, nil
		}
		sliceExists, _ := checkSliceExist(streamInfo.FileHash, slice.SliceStorageInfo.SliceHash)
		if !sliceExists {
			return false, nil
		}
	}

	return true, streamInfo
}

func streamVideoInfoHttp(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ownerWalletAddress, fileHash, err := parseFilePath(req.RequestURI)
	if err != nil {
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	walletSign, reqTime, err := getWalletSignFromLocal(req, fileHash)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	sdmPath := fwtypes.DataMeshId{
		Owner: ownerWalletAddress,
		Hash:  fileHash,
	}.String()

	r := reqDownloadMsg(sdmPath, walletSign, reqTime)
	res := namespace.RpcPubApi().RequestVideoDownload(ctx, r)

	if res.Return != rpc_api.DOWNLOAD_OK {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "failed to get file storage info").ToBytes())
		return
	}

	streamInfo, _, err := getStreamInfo(ctx, fileHash, res.ReqId)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	ret, _ := json.Marshal(streamInfo)
	_, _ = w.Write(ret)
}

func GetVideoSliceCache(w http.ResponseWriter, req *http.Request) {
	streamVideoP2PHelper(w, req)
}

func streamVideoP2P(w http.ResponseWriter, req *http.Request) {
	streamVideoP2PHelper(w, req)
}

func streamVideoP2PHelper(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	if setting.State == msgtypes.PP_ACTIVE {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "Current node is activated and is not allowed to cache video").ToBytes())
		return
	}

	reqPath := req.URL.Path
	pathParams := strings.Split(reqPath, "/")
	if len(pathParams) < 4 {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "bad request").ToBytes())
		return
	}

	reqId := pathParams[2]
	segment := pathParams[3]

	value, ok := RequestInfoMap.Load(reqId)
	if !ok {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "session expired").ToBytes())
		return
	}

	var streamInfo *StreamInfo
	streamInfo = value.(*StreamInfo)

	sliceInfo, ok := streamInfo.SegmentToSliceInfo[segment]
	if !ok {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "unable to find segment's info").ToBytes())
		return
	}

	utils.DebugLog("Send request to retrieve the slice ", sliceInfo.SliceStorageInfo.SliceHash)

	data, err := getSliceData(ctx, streamInfo.FileHash, reqId, sliceInfo)
	if err != nil {
		utils.ErrorLog("failed to get video slice ", err)
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "failed to get video slice").ToBytes())
		return
	}
	if segment == streamInfo.HeaderFile {
		w.Header().Set("Content-Type", "application/x-mpegURL")
	} else if segment != "" {
		w.Header().Set("Content-Type", "video/MP2T")
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
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

		if ppInfo.P2PAddress != p2pserver.GetP2pServer(req.Context()).GetP2PAddress().String() {
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

func parseGetOZoneWalletAddress(reqURL *url.URL) string {
	reqPath := reqURL.Path
	return reqPath[strings.LastIndex(reqPath, "/")+1:]
}

func parseFileHash(reqURL *url.URL) string {
	reqPath := reqURL.Path
	return reqPath[strings.LastIndex(reqPath, "/")+1:]
}

func parseSliceHash(reqURL *url.URL) string {
	reqPath := reqURL.Path
	return reqPath[strings.LastIndex(reqPath, "/")+1:]
}

func cacheVideoSlices(ctx context.Context, streamInfo *StreamInfo, reqId string, twoSlicesReadyCh chan<- bool) {
	slices := getVideoSlicesInfoSortedByName(streamInfo)
	cacheCh := make(chan bool, setting.StreamCacheMaxSlice)

	for i := 0; i < setting.StreamCacheMaxSlice; i++ {
		cacheCh <- true
	}

	for idx, sliceInfo := range slices {
		<-cacheCh
		go func(idx int, sliceInfo *protos.DownloadSliceInfo) {
			exist, _ := checkSliceExist(streamInfo.FileHash, sliceInfo.SliceStorageInfo.SliceHash)
			if !exist {
				_, _ = getSliceData(ctx, streamInfo.FileHash, reqId, sliceInfo)
			}
			if idx < len(slices)-setting.StreamCacheMaxSlice {
				cacheCh <- true
			}
			if idx == 1 {
				twoSlicesReadyCh <- true
			}
		}(idx, sliceInfo)
	}
	close(cacheCh)
}

func getVideoSlicesInfoSortedByName(streamInfo *StreamInfo) []*protos.DownloadSliceInfo {
	var sliceKeys []string
	for key := range streamInfo.SegmentToSliceInfo {
		sliceKeys = append(sliceKeys, key)
	}

	sort.Slice(sliceKeys, func(i, j int) bool {
		if sliceKeys[i] == streamInfo.HeaderFile {
			return true
		}
		if sliceKeys[j] == streamInfo.HeaderFile {
			return false
		}
		fileNameWithoutExt := func(fileName string) string {
			return strings.TrimSuffix(fileName, filepath.Ext(fileName))
		}
		filename1 := fileNameWithoutExt(sliceKeys[i])
		filename2 := fileNameWithoutExt(sliceKeys[j])

		num1, err1 := strconv.Atoi(filename1)
		num2, err2 := strconv.Atoi(filename2)
		if err1 != nil || err2 != nil {
			return filename1 < filename2
		}
		return num1 < num2
	})

	var slices []*protos.DownloadSliceInfo
	for _, key := range sliceKeys {
		slices = append(slices, streamInfo.SegmentToSliceInfo[key])
	}
	return slices
}

func checkSliceExist(fileHash, sliceHash string) (bool, string) {
	slicePath := getSlicePath(fileHash, sliceHash)
	return file.CheckFilePathEx(slicePath), slicePath
}

func checkStreamInfoExist(fileLink, walletAddress string) (bool, string) {
	streamInfoPath := getStreamInfoPath(fileLink, walletAddress)
	return file.CheckFilePathEx(streamInfoPath), streamInfoPath
}

func getSlicePath(folderName, sliceHash string) string {
	folder := filepath.Join(file.GetTmpDownloadPath(), setting.VideoPath, folderName)
	return filepath.Join(folder, sliceHash)
}

func getStreamInfoPath(fileLink, walletAddress string) string {
	folder := filepath.Join(file.GetTmpDownloadPath(), setting.VideoPath, streamInfoFile, fileLink+"_"+walletAddress)
	return filepath.Join(folder, streamInfoFile)
}

func getStreamInfo(ctx context.Context, fileHash, reqId string) (*StreamInfo, *protos.RspFileStorageInfo, error) {
	var fInfo *protos.RspFileStorageInfo
	if f, ok := task.DownloadFileMap.Load(fileHash + reqId); ok {
		fInfo = f.(*protos.RspFileStorageInfo)
		utils.DebugLog("Received file storage info from sp ", fInfo)
	}

	if fInfo == nil {
		return nil, nil, errors.New("http stream video failed to get file storage info!")
	}

	if !crypto.IsVideoStream(fileHash) {
		return nil, nil, errors.New("the file was not uploaded as video stream")
	}

	hlsInfo, err := getHlsInfo(ctx, fInfo)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get hls info!")
	}
	if hlsInfo == nil {
		return nil, nil, errors.New("failed to get hls info!")
	}
	segmentToSliceInfo := make(map[string]*protos.DownloadSliceInfo, 0)
	for segment := range hlsInfo.SegmentToSlice {
		segmentInfo := getVideoSliceInfo(segment, fInfo, hlsInfo)
		segmentToSliceInfo[segment] = segmentInfo
	}
	streamInfo := &StreamInfo{
		FileHash:           fileHash,
		HeaderFile:         hlsInfo.HeaderFile,
		SegmentToSliceInfo: segmentToSliceInfo,
	}
	return streamInfo, fInfo, nil
}

func cacheStreamInfo(fileLink, walletAddress string, streamInfo *StreamInfo) error {
	SegmentToSliceInfo := make(map[string]*protos.DownloadSliceInfo, len(streamInfo.SegmentToSliceInfo))
	for key, slice := range streamInfo.SegmentToSliceInfo {
		SegmentToSliceInfo[key] = &protos.DownloadSliceInfo{
			SliceStorageInfo: slice.SliceStorageInfo,
			SliceNumber:      slice.SliceNumber,
		}
	}
	cachedStreamInfo := StreamInfo{
		HeaderFile:         streamInfo.HeaderFile,
		FileHash:           streamInfo.FileHash,
		SegmentToSliceInfo: SegmentToSliceInfo,
	}
	streamInfoPath := getStreamInfoPath(fileLink, walletAddress)
	rawData, _ := json.Marshal(cachedStreamInfo)
	fileMg, err := os.OpenFile(streamInfoPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		fileMg, err = file.CreateFolderAndReopenFile(filepath.Dir(streamInfoPath), filepath.Base(streamInfoPath))
		if err != nil {
			return err
		}
	}
	defer func() {
		_ = fileMg.Close()
	}()
	_ = file.WriteFile(rawData, 0, fileMg)
	return nil
}

func getHlsInfo(ctx context.Context, fInfo *protos.RspFileStorageInfo) (*file.HlsInfo, error) {
	sliceInfo := getSliceInfoBySliceNumber(fInfo, uint64(1))
	data, err := getSliceData(ctx, fInfo.FileHash, fInfo.ReqId, sliceInfo)
	if err != nil {
		return nil, err
	}
	return file.LoadHlsInfoFromData(data)
}

func getSliceData(ctx context.Context, fileHash, reqId string, sliceInfo *protos.DownloadSliceInfo) ([]byte, error) {
	exist, slicePath := checkSliceExist(fileHash, sliceInfo.SliceStorageInfo.SliceHash)
	if exist {
		data, err := file.GetWholeFileData(slicePath)
		if err == nil {
			return data, nil
		}
	}

	r := reqDownloadDataMsg(fileHash, reqId, sliceInfo)
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

func getWalletSignFromRequest(req *http.Request, keyword string) (*rpc_api.Signature, int64, error) {
	body, err := verifyStreamInfoReqBody(req)
	if err != nil {
		return nil, 0, errors.New("failed to parse request body")
	}

	if body.ReqTime == 0 || body.PubKey == "" || body.Signature == "" {
		return nil, 0, errors.New("invalid reqTime / pubKey / signature")
	}

	sig := rpc_api.Signature{
		Address:   body.WalletAddress,
		Pubkey:    body.PubKey,
		Signature: body.Signature,
	}
	return &sig, body.ReqTime, nil
}

func getWalletSignFromLocal(req *http.Request, keyword string) (*rpc_api.Signature, int64, error) {
	sn, err := handleGetOzone(req.Context(), setting.WalletAddress)
	if err != nil {
		return nil, 0, err
	}
	nowSec := time.Now().Unix()
	sign, err := setting.WalletPrivateKey.Sign([]byte(msgutils.GetFileDownloadWalletSignMessage(keyword, setting.WalletAddress, sn, nowSec)))
	if err != nil {
		return nil, 0, err
	}
	walletPublicKey, err := fwtypes.WalletPubKeyToBech32(setting.WalletPublicKey)
	if err != nil {
		return nil, 0, err
	}
	return &rpc_api.Signature{
		Address:   setting.WalletAddress,
		Pubkey:    walletPublicKey,
		Signature: hex.EncodeToString(sign),
	}, nowSec, nil
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

	if reqBody.FileReqId == "" {
		return nil, errors.Wrap(err, "incorrect file request id")
	}

	return &reqBody, nil
}

func verifyStreamInfoReqBody(req *http.Request) (*StreamInfoRequest, error) {
	body, err := io.ReadAll(req.Body)
	defer req.Body.Close()

	if err != nil {
		return nil, err
	}

	var reqBody StreamInfoRequest
	err = json.Unmarshal(body, &reqBody)

	if err != nil {
		return nil, err
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

	p2pPubKey, err := fwtypes.P2PPubKeyFromBech32(spInfo.P2PPublicKey)
	if err != nil {
		utils.ErrorLog("Error when trying to decode P2P pubKey bech32", err)
		return false
	}

	msg := []byte(reqBody.P2PAddress + reqBody.FileHash + header.ReqDownloadSlice.Name)
	if !p2pPubKey.VerifySignature(msg, reqBody.Sign) {
		return false
	}

	newSlashHash, err := crypto.CalcSliceHash(data, reqBody.FileHash, reqBody.SliceInfo.SliceNumber)
	if err != nil {
		utils.ErrorLog(err)
		return false
	}
	return sliceHash == newSlashHash
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

func reqDownloadMsg(sdmPath string, walletSign *rpc_api.Signature, nowSec int64) rpc_api.ParamReqDownloadFile {
	return rpc_api.ParamReqDownloadFile{
		FileHandle: sdmPath,
		Signature:  *walletSign,
		ReqTime:    nowSec,
	}
}

func reqDownloadDataMsg(fileHash, reqId string, sliceInfo *protos.DownloadSliceInfo) rpc_api.ParamReqDownloadData {
	return rpc_api.ParamReqDownloadData{
		FileHash:       fileHash,
		ReqId:          reqId,
		SliceHash:      sliceInfo.SliceStorageInfo.SliceHash,
		SliceNumber:    sliceInfo.SliceNumber,
		SliceSize:      sliceInfo.SliceStorageInfo.SliceSize,
		NetworkAddress: sliceInfo.StoragePpInfo.NetworkAddress,
		P2PAddress:     sliceInfo.StoragePpInfo.P2PAddress,
	}
}

func reqGetSharedMsg(shareLink pptypes.GetShareFile, walletSign *rpc_api.Signature, nowSec int64) rpc_api.ParamReqGetShared {
	return rpc_api.ParamReqGetShared{
		Signature: *walletSign,
		ReqTime:   nowSec,
		ShareLink: shareLink.String(),
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
