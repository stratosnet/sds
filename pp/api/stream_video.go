package api

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/pp/types"

	//"github.com/stratosnet/sds/relay"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/datamesh"
	"github.com/stratosnet/sds/utils/httpserv"
	utiltypes "github.com/stratosnet/sds/utils/types"
	//tmed25519 "github.com/tendermint/tendermint/crypto/ed25519"
	utiled25519 "github.com/stratosnet/sds/utils/crypto/ed25519"
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
	SliceInfo     *protos.DownloadSliceInfo
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

func streamVideoInfoCache(w http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	ownerWalletAddress, fileHash, err := parseFilePath(req.RequestURI)
	if err != nil {
		w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	if setting.State == types.PP_ACTIVE {
		w.WriteHeader(setting.FAILCode)
		w.Write(httpserv.NewErrorJson(setting.FAILCode, "Current node is activated and is not allowed to cache video").ToBytes())
		return
	}

	task.VideoCacheTaskMap.Delete(fileHash)
	task.DownloadFileMap.Delete(fileHash + task.LOCAL_REQID)
	streamInfo, err := getStreamInfo(ctx, fileHash, ownerWalletAddress, w)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}
	dTask, ok := task.GetDownloadTask(fileHash, setting.WalletAddress, task.LOCAL_REQID)
	if !ok {
		w.WriteHeader(setting.FAILCode)
		w.Write(httpserv.NewErrorJson(setting.FAILCode, "Failed to retrieve download task info").ToBytes())
		return
	}
	event.GetVideoSlices(ctx, streamInfo.FileInfo, dTask)
	ret, _ := json.Marshal(streamInfo)
	w.Write(ret)
}

func streamVideoInfoHttp(w http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	ownerWalletAddress, fileHash, err := parseFilePath(req.RequestURI)
	if err != nil {
		w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	task.DownloadFileMap.Delete(fileHash + task.LOCAL_REQID)
	streamInfo, err := getStreamInfo(ctx, fileHash, ownerWalletAddress, w)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	ret, _ := json.Marshal(streamInfo)
	w.Write(ret)
}

func streamVideoP2P(w http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	sliceHash := parseSliceHash(req.URL)

	body, err := verifyStreamReqBody(req)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	if setting.State == types.PP_ACTIVE {
		w.WriteHeader(setting.FAILCode)
		w.Write(httpserv.NewErrorJson(setting.FAILCode, "Current node is activated and is not allowed to cache video").ToBytes())
		return
	}

	utils.DebugLog("Send request to retrieve the slice ", sliceHash)

	fInfo := &protos.RspFileStorageInfo{
		FileHash:      body.FileHash,
		SavePath:      body.SavePath,
		FileName:      body.FileName,
		NodeSign:      body.Sign,
		SpP2PAddress:  body.SpP2pAddress,
		WalletAddress: setting.WalletAddress,
	}

	event.GetVideoSlice(ctx, body.SliceInfo, fInfo, w)
}

func streamVideoHttp(w http.ResponseWriter, req *http.Request) {
	sliceHash := parseSliceHash(req.URL)

	body, err := verifyStreamReqBody(req)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	utils.DebugLog("Redirect the request to resource node.")
	redirectToResourceNode(body.FileHash, sliceHash, body.RestAddress, setting.WalletAddress, w, req)
	sendReportStreamResult(body, sliceHash, false)
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
	event.ClearFileInfoAndDownloadTask(parseFileHash(req.URL), task.LOCAL_REQID, w)
}

func GetVideoSlice(w http.ResponseWriter, req *http.Request) {
	sliceHash := parseSliceHash(req.URL)

	utils.Log("Received get video slice request :", req.URL)

	body, err := verifyStreamReqBody(req)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	utils.DebugLog("The request body is ", body)

	if dlTask, ok := task.DownloadTaskMap.Load(body.FileHash + setting.WalletAddress + task.LOCAL_REQID); ok {
		utils.DebugLog("Found task info ", body)
		downloadTask := dlTask.(*task.DownloadTask)
		ppInfo := downloadTask.SliceInfo[sliceHash].StoragePpInfo
		if ppInfo.P2PAddress != setting.P2PAddress {
			utils.DebugLog("Current P2PAddress does not have the requested slice")
			targetAddress := ppInfo.RestAddress
			redirectURL := fmt.Sprintf("http://%s/videoSlice/%s", targetAddress, sliceHash)
			utils.DebugLog("Redirect the request to " + redirectURL)
			http.Redirect(w, req, redirectURL, http.StatusTemporaryRedirect)
			return
		}
	}

	utils.DebugLog("Start getting the slice from local storage", body)

	video := file.GetSliceData(sliceHash)
	if video == nil {
		w.WriteHeader(setting.FAILCode)
		w.Write(httpserv.NewErrorJson(setting.FAILCode, "Could not find the video segment!").ToBytes())
		return
	}

	if !verifySignature(body, sliceHash, video) {
		w.WriteHeader(setting.FAILCode)
		w.Write(httpserv.NewErrorJson(setting.FAILCode, "Authorization failed!").ToBytes())
		return
	}

	utils.DebugLog("Found the slice and return", body)
	w.Write(video)

	sendReportStreamResult(body, sliceHash, true)
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

func parseFileHash(reqURL *url.URL) string {
	reqPath := reqURL.Path
	return reqPath[strings.LastIndex(reqPath, "/")+1:]
}

func parseSliceHash(reqURL *url.URL) string {
	reqPath := reqURL.Path
	return reqPath[strings.LastIndex(reqPath, "/")+1:]
}

func getStreamInfo(ctx context.Context, fileHash, ownerWalletAddress string, w http.ResponseWriter) (*StreamInfo, error) {
	filePath := datamesh.DataMeshId{
		Owner: ownerWalletAddress,
		Hash:  fileHash,
	}.String()
	event.GetFileStorageInfo(ctx, filePath, setting.VIDEOPATH, uuid.New().String(), "", true, w)
	var fInfo *protos.RspFileStorageInfo
	start := time.Now().Unix()
	for {
		if f, ok := task.DownloadFileMap.Load(fileHash + task.LOCAL_REQID); ok {
			fInfo = f.(*protos.RspFileStorageInfo)
			utils.DebugLog("Received file storage info from sp ", fInfo)
			break
		} else {
			select {
			case <-time.After(time.Second):
			}
			// timeout
			if time.Now().Unix()-start > setting.HTTPTIMEOUT {
				return nil, errors.New("http stream video failed to get file storage info!")
			}
		}
	}

	hlsInfo := event.GetHlsInfo(ctx, fInfo)
	segmentToSliceInfo := make(map[string]*protos.DownloadSliceInfo, 0)
	for segment := range hlsInfo.SegmentToSlice {
		segmentInfo := event.GetVideoSliceInfo(ctx, segment, fInfo)
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
	body, err := ioutil.ReadAll(req.Body)
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

	if !ed25519.Verify(p2pPubKey.Bytes(), []byte(reqBody.P2PAddress+reqBody.FileHash+header.ReqDownloadSlice), reqBody.Sign) {
		return false
	}

	return sliceHash == utils.CalcSliceHash(data, reqBody.FileHash, reqBody.SliceInfo.SliceNumber)
}

func sendReportStreamResult(body *StreamReqBody, sliceHash string, isPP bool) {
	event.SendReportStreamingResult(&protos.RspDownloadSlice{
		SliceInfo:     &protos.SliceOffsetInfo{SliceHash: sliceHash},
		FileHash:      body.FileHash,
		WalletAddress: setting.WalletAddress,
		P2PAddress:    body.P2PAddress,
		TaskId:        body.SliceInfo.TaskId,
		SpP2PAddress:  body.SpP2pAddress,
	}, isPP)
}
