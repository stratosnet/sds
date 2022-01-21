package api

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	tmed25519 "github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/bech32"

	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/relay/stratoschain"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/httpserv"
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

func streamVideoStorageInfo(w http.ResponseWriter, req *http.Request) {
	url := req.URL.Path
	fileHash := url[strings.LastIndex(url, "/")+1:]

	var fInfo *protos.RspFileStorageInfo
	task.DownloadFileMap.Delete(fileHash)
	event.GetFileStorageInfo("sdm://"+setting.WalletAddress+"/"+fileHash, setting.VIDEOPATH, uuid.New().String(), true, w)
	start := time.Now().Unix()
	for {
		if f, ok := task.DownloadFileMap.Load(fileHash); ok {
			fInfo = f.(*protos.RspFileStorageInfo)
			utils.DebugLog("Received file storage info from sp ", fInfo)
			break
		} else {
			select {
			case <-time.After(time.Second):
			}
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
			HeaderFile:         hlsInfo.HeaderFile,
			SegmentToSliceInfo: segmentToSliceInfo,
			FileInfo:           fInfo,
		})
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

	if setting.State != types.PP_ACTIVE && setting.Config.StreamingCache {
		utils.DebugLog("Send request to sp to retrieve the slice ", sliceHash)

		fInfo := &protos.RspFileStorageInfo{
			FileHash: body.FileHash,
			SavePath: body.SavePath,
			FileName: body.FileName,
		}

		event.GetVideoSlice(body.SliceInfo, fInfo, w)
	} else {
		utils.DebugLog("Redirect the request to resource node.")
		redirectToResourceNode(body.FileHash, sliceHash, body.RestAddress, w, req)
		sendReportStreamResult(body, sliceHash, false)
	}
}

func redirectToResourceNode(fileHash, sliceHash, restAddress string, w http.ResponseWriter, req *http.Request) {
	var targetAddress string
	if dlTask, ok := task.DownloadTaskMap.Load(fileHash + setting.WalletAddress); ok {
		//self is the resource PP and has task info
		downloadTask := dlTask.(*task.DownloadTask)
		targetAddress = downloadTask.SliceInfo[sliceHash].StoragePpInfo.RestAddress
	} else {
		//to ask resource pp for slice addresses
		targetAddress = restAddress
	}
	url := fmt.Sprintf("http://%s/videoSlice/%s", targetAddress, sliceHash)
	utils.DebugLog("Redirect URL ", url)
	http.Redirect(w, req, url, http.StatusTemporaryRedirect)
}

func clearStreamTask(w http.ResponseWriter, req *http.Request) {
	url := req.URL.Path
	fileHash := url[strings.LastIndex(url, "/")+1:]
	event.ClearFileInfoAndDownloadTask(fileHash, w)
}

func GetVideoSlice(w http.ResponseWriter, req *http.Request) {
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
			targetAddress := ppInfo.RestAddress
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

	if len(reqBody.FileHash) != 40 {
		return nil, errors.New("incorrect file fileHash")
	}

	if len(reqBody.WalletAddress) != 41 {
		return nil, errors.New("incorrect wallet address")
	}

	if len(reqBody.P2PAddress) != 47 {
		return nil, errors.New("incorrect P2P address")
	}

	if reqBody.FileName == "" {
		return nil, errors.New("please give file name")
	}

	if len(reqBody.SpP2pAddress) != 47 {
		return nil, errors.New("incorrect SP P2P address")
	}

	if reqBody.RestAddress == "" {
		return nil, errors.New("please give correct rest address to the file slice")
	}

	return &reqBody, nil
}

func verifySignature(reqBody *StreamReqBody, sliceHash string, data []byte) bool {
	if sliceHash != utils.CalcSliceHash(data, reqBody.FileHash) {
		return false
	}
	if val, ok := setting.SPMap.Load(reqBody.SpP2pAddress); ok {
		spInfo, ok := val.(setting.SPBaseInfo)
		if !ok {
			return false
		}
		_, pubKeyRaw, err := bech32.DecodeAndConvert(spInfo.P2PPublicKey)
		if err != nil {
			utils.ErrorLog("Error when trying to decode P2P pubKey bech32", err)
			return false
		}

		p2pPubKey := tmed25519.PubKeyEd25519{}
		err = stratoschain.Cdc.UnmarshalBinaryBare(pubKeyRaw, &p2pPubKey)
		if err != nil {
			utils.ErrorLog("Error when trying to read P2P pubKey ed25519 binary", err)
			return false
		}

		return ed25519.Verify(p2pPubKey[:], []byte(reqBody.P2PAddress+reqBody.FileHash), reqBody.Sign)
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
		SpP2PAddress:  body.SpP2pAddress,
	}, isPP)
}
