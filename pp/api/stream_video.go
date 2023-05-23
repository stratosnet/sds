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
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	rpctypes "github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/rpc"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/pp/types"

	"github.com/stratosnet/sds/utils"
	utiled25519 "github.com/stratosnet/sds/utils/crypto/ed25519"
	"github.com/stratosnet/sds/utils/datamesh"
	"github.com/stratosnet/sds/utils/httpserv"
	utiltypes "github.com/stratosnet/sds/utils/types"
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
	reqId := task.LOCAL_REQID + uuid.New().String()
	ctx := core.RegisterRemoteReqId(req.Context(), reqId)
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
	res := rpc.RpcPubApi().RequestDownload(ctx, r)

	if res.Return != rpctypes.DL_OK_ASK_INFO {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "failed to get file storage info").ToBytes())
		return
	}

	streamInfo, err := getStreamInfo(ctx, fileHash, reqId)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	ret, _ := json.Marshal(streamInfo)
	_, _ = w.Write(ret)
}

func streamSharedVideoInfoCache(w http.ResponseWriter, req *http.Request) {
	reqId := task.LOCAL_REQID + uuid.New().String()
	ctx := core.RegisterRemoteReqId(req.Context(), reqId)

	if setting.State == types.PP_ACTIVE {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "Current node is activated and is not allowed to cache video").ToBytes())
		return
	}

	shareLink, _, _ := parseShareLink(req.RequestURI)
	r := reqGetSharedMsg(shareLink)
	res := rpc.RpcPubApi().RequestGetShared(ctx, r)

	if res.Return != rpctypes.DL_OK_ASK_INFO {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "failed to get file storage info").ToBytes())
		return
	}

	streamInfo, err := getStreamInfo(ctx, res.FileHash, reqId)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	ret, _ := json.Marshal(streamInfo)
	_, _ = w.Write(ret)
}

func streamVideoInfoHttp(w http.ResponseWriter, req *http.Request) {
	reqId := task.LOCAL_REQID + uuid.New().String()
	ctx := core.RegisterRemoteReqId(context.Background(), reqId)
	ownerWalletAddress, fileHash, err := parseFilePath(req.RequestURI)
	if err != nil {
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	task.DownloadFileMap.Delete(fileHash + reqId)

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
	res := rpc.RpcPubApi().RequestDownload(ctx, r)

	if res.Return != rpctypes.DL_OK_ASK_INFO {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "failed to get file storage info").ToBytes())
		return
	}

	streamInfo, err := getStreamInfo(ctx, fileHash, reqId)
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

	r := reqDownloadDataMsg(fInfo, body.SliceInfo, body.FileReqId)
	res := rpc.RpcPubApi().RequestDownloadData(ctx, r)

	if res.Return != rpctypes.DOWNLOAD_OK {
		w.WriteHeader(setting.FAILCode)
		_, _ = w.Write(httpserv.NewErrorJson(setting.FAILCode, "failed to get video slice").ToBytes())
		return
	}

	decoded, _ := base64.StdEncoding.DecodeString(res.FileData)
	w.Write(decoded)
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
		targetAddress = downloadTask.SliceInfo[sliceHash].StoragePpInfo.RestAddress
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
		downloadTask := dlTask.(*task.DownloadTask)
		ppInfo := downloadTask.SliceInfo[sliceHash].StoragePpInfo
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

	fInfo.ReqId = reqId
	hlsInfo := getHlsInfo(ctx, fInfo, reqId)
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

	if _, err := utiltypes.P2pAddressFromBech(reqBody.P2PAddress); err != nil {
		return nil, errors.Wrap(err, "incorrect P2P address")
	}

	if reqBody.FileName == "" {
		return nil, errors.New("please give file name")
	}

	if _, err := utiltypes.P2pAddressFromBech(reqBody.SpP2pAddress); err != nil {
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

	p2pPubKey := utiled25519.PubKeyBytesToPubKey(pubKeyRaw)
	msg := []byte(reqBody.P2PAddress + reqBody.FileHash + header.ReqDownloadSlice)
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

func getHlsInfo(ctx context.Context, fInfo *protos.RspFileStorageInfo, reqId string) *file.HlsInfo {
	sliceInfo := getSliceInfoBySliceNumber(fInfo, uint64(1))
	sliceHash := sliceInfo.SliceStorageInfo.SliceHash

	r := reqDownloadDataMsg(fInfo, sliceInfo, reqId)
	res := rpc.RpcPubApi().RequestDownloadData(ctx, r)

	if res.Return != rpctypes.DOWNLOAD_OK {
		return nil
	}

	if file.CheckSliceExisting(fInfo.FileHash, fInfo.FileName, sliceHash, fInfo.SavePath, fInfo.ReqId) {
		return file.LoadHlsInfo(fInfo.FileHash, sliceHash, fInfo.SavePath)
	}
	return nil
}

func getVideoSliceInfo(sliceName string, fInfo *protos.RspFileStorageInfo, hlsInfo *file.HlsInfo) *protos.DownloadSliceInfo {
	var sliceNumber uint64
	sliceNumber = hlsInfo.SegmentToSlice[sliceName]
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

func reqDownloadMsg(hash, sdmPath, sn string) rpctypes.ParamReqDownloadFile {
	// param
	sign, _ := utiltypes.BytesToAccPriveKey(setting.WalletPrivateKey).Sign([]byte(utils.GetFileDownloadWalletSignMessage(hash, setting.WalletAddress, sn)))

	return rpctypes.ParamReqDownloadFile{
		FileHandle:   sdmPath,
		WalletAddr:   setting.WalletAddress,
		WalletPubkey: string(setting.WalletPublicKey),
		Signature:    hex.EncodeToString(sign),
	}
}

func reqDownloadDataMsg(fInfo *protos.RspFileStorageInfo, sliceInfo *protos.DownloadSliceInfo, reqId string) rpctypes.ParamReqDownloadData {
	return rpctypes.ParamReqDownloadData{
		FileHash:       fInfo.FileHash,
		ReqId:          reqId,
		SliceNumber:    sliceInfo.SliceNumber,
		NetworkAddress: sliceInfo.StoragePpInfo.NetworkAddress,
		P2PAddress:     sliceInfo.StoragePpInfo.P2PAddress,
		WalletAddr:     fInfo.WalletAddress,
	}
}

func reqGetSharedMsg(shareLink string) rpctypes.ParamReqGetShared {
	return rpctypes.ParamReqGetShared{
		WalletAddr:   setting.WalletAddress,
		WalletPubkey: string(setting.WalletPublicKey),
		ShareLink:    shareLink,
	}
}

func handleGetOzone(ctx context.Context, walletAddress string) (string, error) {
	utils.Log("- request ozone balance (method: user_requestGetOzone)")
	res := rpc.RpcPubApi().RequestGetOzone(ctx, rpctypes.ParamReqGetOzone{
		WalletAddr: walletAddress,
	})

	if res.Return == rpctypes.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
		ozone, _ := strconv.ParseFloat(res.Ozone, 64)
		utils.Log("OZONE balance: ", ozone/1000000000.0)
		utils.Log("SN:            ", res.SequenceNumber)
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}

	return res.SequenceNumber, nil
}
