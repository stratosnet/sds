package event

import (
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/httpserv"
)

// HTTPType
type HTTPType int

//
const (
	HTTPGetAllFile HTTPType = 1 + iota
	HTTPDownloadFile
	HTTPUploadFile
	HTTPDeleteFile
	HTTPShareLink
	HTTPShareFile
	HTTPDeleteShare
	HTTPGetShareFile
	HTTPDownloadSlice
)

// HTTPRsp HTTPRsp
type HTTPRsp struct {
	Data  interface{}
	Type  HTTPType
	ReqID string
}

type allFile struct {
	FileSize           uint64 `json:"fileSize"`
	FileHash           string `json:"fileHash"`
	FileName           string `json:"fileName"`
	CreateTime         uint64 `json:"createTime"`
	IsDirectory        bool   `json:"isDirectory"`
	IsPrivate          bool   `json:"isPrivate"`
	OwnerWalletAddress string `json:"ownerWalletAddress"`
	ShareLink          string `json:"shareLink"`
	ID                 uint64 `json:"id"`
	StoragePath        string `json:"storagePath"`
}

// HTTPRspMap
var HTTPRspMap = &sync.Map{}

// HTTPWriterMap
var HTTPWriterMap = &sync.Map{}

func putData(reqID string, httpType HTTPType, target interface{}) {
	if setting.Config.InternalPort != "" && setting.WalletAddress != "" {
		// httpRsp(target.ReqId, HTTPDownloadFile, &target)
		rsp := &HTTPRsp{
			Data: target,
			Type: httpType,
		}
		HTTPRspMap.Store(reqID, rsp)
	}
}

func storeResponseWriter(reqID string, w http.ResponseWriter) error {
	if setting.Config.InternalPort != "" && setting.WalletAddress != "" {
		if w != nil {
			return StoreReqID(reqID, w)
		}
	}
	return nil
}

func notLogin(w http.ResponseWriter) {
	if w != nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "login first").ToBytes())
	}
}

// HTTPStartListen HTTPStartListen
func HTTPStartListen(reqID string) error {
	start := time.Now().Unix()
	for {
		var httpRsp *HTTPRsp
		var write http.ResponseWriter

		if d, ok := HTTPRspMap.Load(reqID); ok {
			httpRsp = d.(*HTTPRsp)
		} else {
			// timeout
			if time.Now().Unix()-start > setting.HTTPTIMEOUT {
				utils.DebugLog("failed to get reqId!")
				return errors.New("time out for reqId " + reqID)
			}
			continue
		}

		if w, ok := HTTPWriterMap.Load(reqID); ok {
			write = w.(http.ResponseWriter)
		} else {
			HTTPWriterMap.Delete(reqID)
			HTTPRspMap.Delete(reqID)
			return errors.New("could not find ResponseWriter for reqId " + reqID)
		}

		switch httpRsp.Type {
		case HTTPDownloadFile:
			{
				HTTPDownloadFileFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPDownloadSlice:
			{
				HTTPDownloadSliceFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPGetAllFile:
			{
				HTTPGetAllFileFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPDeleteFile:
			{
				HTTPDeleteFileFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPShareLink:
			{
				HTTPShareLinkFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPShareFile:
			{
				HTTPShareFileFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPDeleteShare:
			{
				HTTPDeleteShareFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPGetShareFile:
			{
				HTTPGetShareFileFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPUploadFile:
			{
				HTTPUploadFileFun(httpRsp, write, reqID)
				return nil
			}
		}
	}
}

// HTTPUploadFileFun HTTPUploadFileFun
func HTTPUploadFileFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	type upLoadFileResult struct {
		TaskID             string `json:"taskID"`
		FileName           string `json:"fileName"`
		FileHash           string `json:"fileHash"`
		ImageWalletAddress string `json:"imageWalletAddress"`
	}
	target := httpRsp.Data.(*protos.RspUploadFile)
	utils.DebugLog("result>>>>>>>>>>>>>>", target)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		result := make(map[string]*upLoadFileResult, 0)
		r := &upLoadFileResult{
			TaskID:             target.ReqId,
			FileHash:           target.FileHash,
			ImageWalletAddress: target.OwnerWalletAddress,
		}
		setting.UploadTaskIDMap.Store(r.TaskID, target.FileHash)
		result["cover"] = r
		utils.DebugLog("cover?>>>>>>>>>", result)
		write.Write(httpserv.NewJson(result, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPUploadFileFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPGetShareFileFun HTTPGetShareFileFun
func HTTPGetShareFileFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspGetShareFile)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		list := make([]*allFile, 0)
		for _, f := range target.FileInfo {
			file := &allFile{
				FileHash:           f.FileHash,
				FileName:           f.FileName,
				FileSize:           f.FileSize,
				CreateTime:         f.CreateTime,
				IsDirectory:        f.IsDirectory,
				IsPrivate:          f.IsPrivate,
				OwnerWalletAddress: f.OwnerWalletAddress,
				ShareLink:          f.ShareLink,
			}
			list = append(list, file)
		}
		data := make(map[string][]*allFile, 0)
		data["fileList"] = list
		write.Write(httpserv.NewJson(data, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	if target.IsPrivate {
		utils.DebugLog("HTTPGetShareFileFun error: target is private ")
		write.Write(httpserv.NewJson(nil, setting.ShareErrorCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPGetShareFileFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPDeleteShareFun HTTPDeleteShareFun
func HTTPDeleteShareFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspDeleteShare)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		write.Write(httpserv.NewJson(nil, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPDeleteShareFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPShareFileFun HTTPShareFileFun
func HTTPShareFileFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	type share struct {
		SharePassword string `json:"sharePassword"`
		ShareLink     string `json:"shareLink"`
		ShareID       string `json:"shareID"`
	}
	target := httpRsp.Data.(*protos.RspShareFile)
	if target.Result.State == protos.ResultState_RES_SUCCESS {

		shareFile := &share{
			ShareLink:     target.ShareLink,
			ShareID:       target.ShareId,
			SharePassword: target.SharePassword,
		}
		data := make(map[string]*share, 0)
		data["shareFile"] = shareFile
		write.Write(httpserv.NewJson(data, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPShareFileFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPShareLinkFun HTTPShareLinkFun
func HTTPShareLinkFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	type allShare struct {
		FileSize           uint64 `json:"fileSize"`
		FileHash           string `json:"fileHash"`
		FileName           string `json:"fileName"`
		CreateTime         uint64 `json:"createTime"`
		LinkExpiryTime     uint64 `json:"linkExpiryTime"`
		ShareLink          string `json:"shareLink"`
		ShareLinkPassword  string `json:"shareLinkPassword"`
		ShareID            string `json:"shareID"`
		IsDirectory        bool   `json:"isDirectory"`
		IsPrivate          bool   `json:"isPrivate"`
		OwnerWalletAddress string `json:"ownerWalletAddress"`
	}
	target := httpRsp.Data.(*protos.RspShareLink)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		list := make([]*allShare, 0)
		for _, f := range target.ShareInfo {
			all := &allShare{
				FileHash:           f.FileHash,
				FileSize:           f.FileSize,
				FileName:           f.Name,
				CreateTime:         f.LinkTime,
				LinkExpiryTime:     f.LinkTimeExp,
				ShareID:            f.ShareId,
				IsDirectory:        f.IsDirectory,
				IsPrivate:          f.IsPrivate,
				ShareLink:          f.ShareLink,
				ShareLinkPassword:  f.ShareLinkPassword,
				OwnerWalletAddress: f.OwnerWalletAddress,
			}
			list = append(list, all)
		}

		data := make(map[string][]*allShare, 0)
		data["shareList"] = list
		write.Write(httpserv.NewJson(data, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPShareLinkFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPDeleteFileFun HTTPDeleteFileFun
func HTTPDeleteFileFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspDeleteFile)
	utils.DebugLog("HTTPDeleteFileFun>>>>>", target)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		write.Write(httpserv.NewJson(nil, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPDeleteFileFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPDownloadFileFun HTTPDownloadFileFun
func HTTPDownloadFileFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {

	target := httpRsp.Data.(*protos.RspFileStorageInfo)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		data := make(map[string]interface{})
		var f *os.File
		var err error
		for {
			f, err = os.Open(setting.IMAGEPATH + target.FileHash)
			if err == nil {
				time.Sleep(1 * time.Second)
				goto FILE
			}
		}
	FILE:
		img, err := ioutil.ReadAll(f)
		if err != nil {
			data["image"] = ""
		}
		data["image"] = img

		write.Write(httpserv.NewJson(data, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)

		return
	}
	utils.DebugLog("HTTPDownloadFileFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

func HTTPDownloadSliceFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspDownloadSlice)
	slicePath := file.GetDownloadTmpPath(target.FileHash, target.SliceInfo.SliceHash, target.SavePath)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		video, err := ioutil.ReadFile(slicePath)
		if err != nil {
			write.WriteHeader(setting.FAILCode)
			write.Write(httpserv.NewJson(nil, setting.FAILCode, err.Error()).ToBytes())
		}
		utils.Log("Received video slice: ", target.SliceInfo.SliceHash, "from file: ", target.FileHash)
		write.Write(video)
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPDownloadSliceFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPGetAllFileFun HTTPGetAllFileFun
func HTTPGetAllFileFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspFindMyFileList)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		list := make([]*allFile, 0)
		for _, f := range target.FileInfo {
			all := &allFile{
				FileHash:    f.FileHash,
				FileSize:    f.FileSize,
				FileName:    f.FileName,
				CreateTime:  f.CreateTime,
				IsDirectory: f.IsDirectory,
			}
			list = append(list, all)
		}

		data := make(map[string]interface{}, 0)
		data["fileList"] = list
		write.Write(httpserv.NewJson(data, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// StoreReqID
func StoreReqID(reqID string, w http.ResponseWriter) error {
	HTTPWriterMap.Store(reqID, w)
	return HTTPStartListen(reqID)
}
