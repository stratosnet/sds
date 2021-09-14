package streaming

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stratosnet/sds/utils"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils/httpserv"
)

type StreamReqBody struct {
	FileHash         string
	FileName         string
	WalletAddress    string
	P2PAddress       string
	SpP2pAddress     string
	StreamingAddress string
	Sign             []byte
	SavePath         string
	SliceInfo        *protos.DownloadSliceInfo
}

type StreamInfo struct {
	FileHash           string
	FileName           string
	Sign               []byte
	StreamingAddress   string
	HeaderFile         string
	SavePath           string
	SpP2pAddress       string
	SegmentToSliceInfo map[string]*protos.DownloadSliceInfo
	FileInfo           *protos.RspFileStorageInfo
	HlsInfo            *file.HlsInfo
}

type SliceInfo struct {
	SliceHash string
	TaskId    string
}

func fileStorageInfo(w http.ResponseWriter, req *http.Request) {
	url := req.URL.Path
	fileHash := url[strings.LastIndex(url, "/")+1:]

	var fInfo *protos.RspFileStorageInfo
	event.GetFileStorageInfo("spb://"+setting.WalletAddress+"/"+fileHash, setting.VIDEOPATH, uuid.New().String(), false, true, w)
	start := time.Now().Unix()
	for {
		if f, ok := task.DownloadFileMap.Load(fileHash); ok {
			fInfo = f.(*protos.RspFileStorageInfo)
			utils.DebugLog("Received file storage info from sp ", fInfo)
			break
		} else {
			// timeout
			if time.Now().Unix()-start > setting.HTTPTIMEOUT {
				w.WriteHeader(setting.FAILCode)
				w.Write(httpserv.NewErrorJson(setting.FAILCode, "http stream video failed to get file storage info!").ToBytes())
				return
			}
		}
	}

	hlsInfo := event.GetHlsInfo(fInfo)
	segmentToSliceInfo := make(map[string]*protos.DownloadSliceInfo, 0)
	for segment := range hlsInfo.SegmentToSlice {
		segmentInfo := event.GetVideoSliceInfo(segment, fInfo)
		segmentToSliceInfo[segment] = segmentInfo
	}

	ret, _ := json.Marshal(
		StreamInfo{
			HlsInfo:            hlsInfo,
			HeaderFile:         hlsInfo.HeaderFile,
			Sign:               fInfo.Sign,
			StreamingAddress:   fInfo.StreamingAddress,
			FileHash:           fileHash,
			SavePath:           fInfo.SavePath,
			SpP2pAddress:       fInfo.SpP2PAddress,
			SegmentToSliceInfo: segmentToSliceInfo,
			FileInfo:           fInfo,
		})
	//task.DownloadFileMap.Delete(fileHash)
	w.Write(ret)
}

func streamVideo(w http.ResponseWriter, req *http.Request) {
	url := req.URL.Path
	sliceHash := url[strings.LastIndex(url, "/")+1:]

	body, err := verifyStreamReqBody(req, "")
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	if setting.State != setting.PP_ACTIVE && setting.Config.StreamingCache {
		utils.DebugLog("Send request to sp to retrieve the slice ", sliceHash)

		fInfo := &protos.RspFileStorageInfo{
			FileHash: body.FileHash,
			SavePath: body.SavePath,
			FileName: body.FileName,
		}

		event.GetVideoSlice(body.SliceInfo, fInfo, w)
	} else {
		utils.DebugLog("Redirect the request to resource node.")
		redirectToResourceNode(body.FileHash, sliceHash, body.StreamingAddress, w, req)
		sendReportStreamResult(body, sliceHash, false)
	}
}

func redirectToResourceNode(fileHash, sliceHash, streamingAddress string, w http.ResponseWriter, req *http.Request) {
	var targetAddress string
	if dlTask, ok := task.DownloadTaskMap.Load(fileHash + setting.WalletAddress); ok {
		//self is the resource PP and has task info
		downloadTask := dlTask.(*task.DownloadTask)
		targetAddress = downloadTask.SliceInfo[sliceHash].StoragePpInfo.StreamingAddress
	} else {
		//to ask resource pp for slice addresses
		targetAddress = streamingAddress
	}
	url := fmt.Sprintf("http://%s/videoSlice/%s", targetAddress, sliceHash)
	utils.DebugLog("Redirect URL ", url)
	http.Redirect(w, req, url, http.StatusTemporaryRedirect)
}

func getVideoSlice(w http.ResponseWriter, req *http.Request) {
	url := req.URL.Path
	sliceHash := url[strings.LastIndex(url, "/")+1:]

	utils.Log("Received get video slice request :", url)

	body, err := verifyStreamReqBody(req, sliceHash)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	utils.DebugLog("The request body is ", body)

	if dlTask, ok := task.DownloadTaskMap.Load(body.FileHash + body.WalletAddress); ok {
		utils.DebugLog("Found task info ", body)
		downloadTask := dlTask.(*task.DownloadTask)
		ppInfo := downloadTask.SliceInfo[sliceHash].StoragePpInfo
		if ppInfo.P2PAddress != setting.P2PAddress {
			utils.DebugLog("Current P2PAddress does not have the requested slice")
			targetAddress := ppInfo.StreamingAddress
			url := fmt.Sprintf("http://%s/videoSlice/%s", targetAddress, sliceHash)
			utils.DebugLog("Redirect the request to " + url)
			http.Redirect(w, req, url, http.StatusTemporaryRedirect)
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

func verifyStreamReqBody(req *http.Request, sliceHash string) (*StreamReqBody, error) {
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

	if len(reqBody.FileHash) != 64 {
		return nil, errors.New("incorrect file hash")
	}

	if len(reqBody.WalletAddress) != 41 {
		return nil, errors.New("incorrect wallet address")
	}

	return &reqBody, nil
}

func verifySignature(reqBody *StreamReqBody, sliceHash string, data []byte) bool {
	if sliceHash != utils.CalcSliceHash(data, reqBody.FileHash) {
		return false
	}
	if pubKey, ok := setting.SPPublicKey[reqBody.SpP2pAddress]; ok {
		return ed25519.Verify(pubKey, []byte(reqBody.P2PAddress+reqBody.FileHash), reqBody.Sign)
	} else {
		return false
	}
}

func sendReportStreamResult(body *StreamReqBody, sliceHash string, isPP bool) {
	event.SendReportStreamingResult(&protos.RspDownloadSlice{
		SliceInfo:     &protos.SliceOffsetInfo{SliceHash: sliceHash},
		FileHash:      body.FileHash,
		WalletAddress: body.WalletAddress,
		P2PAddress:    body.P2PAddress,
		TaskId:        body.SliceInfo.TaskId,
	}, isPP)
}
