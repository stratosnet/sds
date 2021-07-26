package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
	"io/ioutil"
	"net/http"
	"path"
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
	FileHash      string
	WalletAddress string
	Token         string
}

func streamVideo(w http.ResponseWriter, req *http.Request) {
	url := req.URL.Path
	segmentName := url[strings.LastIndex(url, "/")+1:]

	body, err := parseStreamReqBody(req)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	utils.Log("Received stream video request :", url)
	utils.DebugLog("The request body is ", body)

	var fInfo *protos.RspFileStorageInfo
	if value, ok := task.DownloadFileMap.Load(body.FileHash); ok {
		//TODO use go routine to clean the map on daily basis
		utils.DebugLog("Found file storage info")
		fInfo = value.(*protos.RspFileStorageInfo)
	} else {
		utils.DebugLog("Could not find file storage info, send request to SP")
		event.GetFileStorageInfo("spb://"+body.WalletAddress+"/"+body.FileHash, setting.VIDEOPATH, uuid.New().String(),
			false, true, w)
		start := time.Now().Unix()
		for {
			if f, ok := task.DownloadFileMap.Load(body.FileHash); ok {
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
	}

	sliceInfo := event.GetVideoSliceInfo(segmentName, fInfo)
	if path.Ext(segmentName) == ".m3u8" || (setting.State != table.PP_ACTIVE && setting.Config.StreamingCache) {
		utils.DebugLog("Send request to sp to retrieve the segment ", segmentName)
		event.GetVideoSlice(sliceInfo, fInfo, w)
	} else {
		utils.DebugLog("Redirect the request to resource node.")
		sliceHash := sliceInfo.SliceStorageInfo.SliceHash
		redirectToResourceNode(body.FileHash, sliceHash, w, req)
	}
}

func redirectToResourceNode(fileHash, sliceHash string, w http.ResponseWriter, req *http.Request) {
	var targetIp string
	if dlTask, ok := task.DownloadTaskMap.Load(fileHash + setting.WalletAddress); ok {
		//self is the resource PP and has task info
		downloadTask := dlTask.(*task.DownloadTask)
		targetIp = getIpFromNetworkAddress(downloadTask.SliceInfo[sliceHash].StoragePpInfo.NetworkAddress)
	} else {
		//to ask resource pp for slice addresses
		if c, ok := client.PDownloadPassageway.Load(fileHash); ok {
			conn := c.(*cf.ClientConn)
			targetIp = conn.GetIP()
		} else {
			conn := client.NewClient(client.PPConn.GetName(), false)
			targetIp = conn.GetIP()
			client.PDownloadPassageway.Store(fileHash, conn)
		}
	}
	url := fmt.Sprintf("http://%s:%d/videoSlice/%s", targetIp, httpserv.API_PORT, sliceHash)
	utils.DebugLog("Redirect URL ", url)
	http.Redirect(w, req, url, http.StatusTemporaryRedirect)
}

func getVideoSlice(w http.ResponseWriter, req *http.Request) {
	url := req.URL.Path
	sliceHash := url[strings.LastIndex(url, "/")+1:]

	utils.Log("Received get video slice request :", url)

	body, err := parseStreamReqBody(req)
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
			targetIp := getIpFromNetworkAddress(ppInfo.NetworkAddress)
			url := fmt.Sprintf("http://%s:%d/videoSlice/%s", targetIp, httpserv.API_PORT, sliceHash)
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

	utils.DebugLog("Found the slice and return", body)
	w.Write(video)
}

func parseStreamReqBody(req *http.Request) (*StreamReqBody, error) {
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

	if !isAuthorized(reqBody) {
		return nil, errors.New("the account does not have access to the file")
	}

	return &reqBody, nil
}

func isAuthorized(reqBody StreamReqBody) bool {
	//TODO evaluate the authorization
	return true
}

func getIpFromNetworkAddress(networkAddress string) string {
	if idx := strings.LastIndex(networkAddress, ":"); idx > -1 {
		return networkAddress[:idx]
	}
	return networkAddress
}
