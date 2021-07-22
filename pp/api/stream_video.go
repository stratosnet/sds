package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/sp/storages/table"
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

	var fInfo *protos.RspFileStorageInfo
	if value, ok := task.DownloadFileMap.Load(body.FileHash); ok {
		//TODO use go routine to clean the map on daily basis
		fInfo = value.(*protos.RspFileStorageInfo)
	} else {
		event.GetFileStorageInfo("spb://"+body.WalletAddress+"/"+body.FileHash, setting.VIDEOPATH, uuid.New().String(),
			false, true, w)
		start := time.Now().Unix()
		for {
			if f, ok := task.DownloadFileMap.Load(body.FileHash); ok {
				fInfo = f.(*protos.RspFileStorageInfo)
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
		event.GetVideoSlice(sliceInfo, fInfo, w)
	} else {
		sliceHash := sliceInfo.SliceStorageInfo.SliceHash
		redirectToResource(body.FileHash, sliceHash, w, req)
	}
}

func redirectToResource(fileHash, sliceHash string, w http.ResponseWriter, req *http.Request) {
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
	url := fmt.Sprintf("http://%s:%d/videoSlice/%s", targetIp, 9609, sliceHash)
	//url := fmt.Sprintf("http://%s:%d/videoSlice/%s/%s", targetIp, 9609, FileHash+setting.WalletAddress, sliceHash)
	http.Redirect(w, req, url, http.StatusTemporaryRedirect)
}

func getVideoSlice(w http.ResponseWriter, req *http.Request) {
	url := req.URL.Path
	sliceHash := url[strings.LastIndex(url, "/")+1:]

	body, err := parseStreamReqBody(req)
	if err != nil {
		w.WriteHeader(setting.FAILCode)
		w.Write(httpserv.NewErrorJson(setting.FAILCode, err.Error()).ToBytes())
		return
	}

	if dlTask, ok := task.DownloadTaskMap.Load(body.FileHash + body.WalletAddress); ok {
		downloadTask := dlTask.(*task.DownloadTask)
		ppInfo := downloadTask.SliceInfo[sliceHash].StoragePpInfo
		if ppInfo.WalletAddress != setting.WalletAddress {
			targetIp := getIpFromNetworkAddress(ppInfo.NetworkAddress)
			url := fmt.Sprintf("http://%s:%d/videoSlice/%s/%s", targetIp, 9609, sliceHash)
			http.Redirect(w, req, url, http.StatusTemporaryRedirect)
			return
		}
	}

	video := file.GetSliceData(sliceHash)
	if video == nil {
		w.WriteHeader(setting.FAILCode)
		w.Write(httpserv.NewErrorJson(setting.FAILCode, "Could not find the video segment!").ToBytes())
		return
	}

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
