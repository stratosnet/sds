package event

import (
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	HTTPMkdir
	HTTPRMdir
	HTTPMVdir
	HTTPShareLink
	HTTPShareFile
	HTTPDeleteShare
	HTTPGetConfig
	HTTPDownPause
	HTTPCreateAlbum
	HTTPFindMyAlbum
	HTTPEditAlbum
	HTTPAlbumContent
	HTTPSearchAlbum
	HTTPGetShareFile
	HTTPInvite
	HTTPReward
	HTTPCollectionAlbum
	HTTPAbstractAlbum
	HTTPMyCollectionAlbum
	HTTPDeleteAlbum
	HTTPSaveFolder
	HTTPGetCapacity
	HTTPFileSort
	HTTPDirectoryTree
	HTTPGetAllDirectory
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

type myAlbum struct {
	AlbumId       string           `json:"albumId"`
	AlbumName     string           `json:"albumName"`
	AlbumBlurb    string           `json:"albumBlurb"`
	AlbumVisit    int64            `json:"albumVisit"`
	AlbumTime     int64            `json:"albumTime"`
	AlbumCover    string           `json:"albumCover"`
	IsPrivate     bool             `json:"isPrivate"`
	IsCollection  bool             `json:"isCollection"`
	WalletAddress string           `json:"walletAddress"`
	AlbumType     protos.AlbumType `json:"albumType"`
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
		case HTTPMkdir:
			{
				HTTPMKdirFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPMVdir:
			{
				HTTPMVdirFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPGetConfig:
			{
				HTTPGetConfigFun(httpRsp, write, reqID)
				return nil
			}
		// case HTTPDownPause:
		// 	{
		// 		HTTPDownPauseFun(httpRsp, write, reqID)
		// 		return nil
		// 	}
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
		case HTTPCreateAlbum:
			{
				HTTPCreateAlbumFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPFindMyAlbum:
			{
				HTTPFindMyAlbumFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPEditAlbum:
			{
				HTTPEditAlbumFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPAlbumContent:
			{
				HTTPAlbumContentFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPSearchAlbum:
			{
				HTTPSearchAlbumFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPGetShareFile:
			{
				HTTPGetShareFileFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPInvite:
			{
				HTTPInviteFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPReward:
			{
				HTTPRewardFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPRMdir:
			{
				HTTPRMdirFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPCollectionAlbum:
			{
				HTTPCollectionAlbumFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPAbstractAlbum:
			{
				HTTPAbstractAlbumFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPMyCollectionAlbum:
			{
				HTTPMyCollectionAlbumFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPDeleteAlbum:
			{
				HTTPDeleteAlbumFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPSaveFolder:
			{
				HTTPSaveFolderFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPGetCapacity:
			{
				HTTPGetCapacityFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPUploadFile:
			{
				HTTPUploadFileFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPFileSort:
			{
				HTTPFileSortFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPDirectoryTree:
			{
				HTTPDirectoryTreeFun(httpRsp, write, reqID)
				return nil
			}
		case HTTPGetAllDirectory:
			{
				HTTPGetAllDirectoryFun(httpRsp, write, reqID)
				return nil
			}
		}
	}
}

// HTTPGetAllDirectoryFun HTTPGetAllDirectoryFun
func HTTPGetAllDirectoryFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspFindDirectory)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		list := make([]*allFile, 0)
		for _, f := range target.FileInfo {
			all := &allFile{
				FileHash:    f.FileHash,
				FileName:    f.FileName,
				StoragePath: f.StoragePath,
				IsDirectory: f.IsDirectory,
			}
			list = append(list, all)
		}

		data := make(map[string]interface{}, 0)
		data["list"] = list
		write.Write(httpserv.NewJson(data, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPGetAllDirectoryFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPDirectoryTreeFun HTTPDirectoryTreeFun
func HTTPDirectoryTreeFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspFindDirectoryTree)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		write.Write(httpserv.NewJson(nil, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPDirectoryTreeFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPFileSortFun HTTPFileSortFun
func HTTPFileSortFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspFileSort)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		write.Write(httpserv.NewJson(nil, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPFileSortFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
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
		setting.UpLoadTaskIDMap.Store(r.TaskID, target.FileHash)
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

// HTTPGetCapacityFun HTTPGetCapacityFun
func HTTPGetCapacityFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	type configData struct {
		FreeCapacity uint64 `json:"FreeCapacity"` // user free capacity, Mb
		Capacity     uint64 `json:"Capacity"`     // total capacity, Mb
	}
	target := httpRsp.Data.(*protos.RspGetCapacity)
	utils.DebugLog("httpRsp?>>>>>>>>>>>>")
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		conf := &configData{
			FreeCapacity: target.FreeCapacity,
			Capacity:     target.Capacity,
		}
		write.Write(httpserv.NewJson(conf, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPGetCapacityFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPSaveFolderFun HTTPSaveFolderFun
func HTTPSaveFolderFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspSaveFolder)
	utils.DebugLog("httpRsp?>>>>>>>>>>>>")
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		write.Write(httpserv.NewJson(nil, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPSaveFolderFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPDeleteAlbumFun HTTPDeleteAlbumFun
func HTTPDeleteAlbumFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspDeleteAlbum)
	utils.DebugLog("httpRsp?>>>>>>>>>>>>")
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		write.Write(httpserv.NewJson(nil, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPDeleteAlbumFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPMyCollectionAlbumFun HTTPMyCollectionAlbumFun
func HTTPMyCollectionAlbumFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspMyCollectionAlbum)
	utils.DebugLog("httpRsp?>>>>>>>>>>>>")
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		list := make([]*myAlbum, 0)
		for _, f := range target.AlbumInfo {
			all := &myAlbum{
				AlbumBlurb: f.AlbumBlurb,
				AlbumId:    f.AlbumId,
				AlbumName:  f.AlbumName,
				AlbumTime:  f.AlbumTime,
				AlbumVisit: f.AlbumVisit,
				AlbumCover: f.AlbumCoverLink,
			}
			list = append(list, all)
		}
		data := make(map[string]interface{}, 0)
		data["albumList"] = list
		write.Write(httpserv.NewJson(data, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPMyCollectionAlbumFun error")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPAbstractAlbumFun HTTPAbstractAlbumFun
func HTTPAbstractAlbumFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	type ab struct {
		All   int64 `json:"all"`
		Video int64 `json:"video"`
		Music int64 `json:"music"`
		Other int64 `json:"other"`
	}
	target := httpRsp.Data.(*protos.RspAbstractAlbum)
	utils.DebugLog("httpRsp?>>>>>>>>>>>>")
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		myAb := make(map[string]*ab, 0)
		ma := &ab{
			All:   target.MyAlbum.All,
			Video: target.MyAlbum.Video,
			Music: target.MyAlbum.Music,
			Other: target.MyAlbum.Other,
		}
		myAb["myAlbum"] = ma
		cAb := &ab{
			All:   target.CollectionAlbum.All,
			Video: target.CollectionAlbum.Video,
			Music: target.CollectionAlbum.Music,
			Other: target.CollectionAlbum.Other,
		}
		myAb["collectionAlbum"] = cAb
		write.Write(httpserv.NewJson(myAb, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPAbstractAlbumFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPCollectionAlbumFun HTTPCollectionAlbumFun
func HTTPCollectionAlbumFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspCollectionAlbum)
	utils.DebugLog("httpRsp?>>>>>>>>>>>>")
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		data := make(map[string]interface{}, 0)
		data["isCollection"] = target.IsCollection
		write.Write(httpserv.NewJson(data, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPCollectionAlbumFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPRMdirFun HTTPRMdirFun
func HTTPRMdirFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspRemoveDirectory)
	utils.DebugLog("httpRsp?>>>>>>>>>>>>")
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		write.Write(httpserv.NewJson(nil, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPRMdirFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPRewardFun HTTPRewardFun
func HTTPRewardFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspGetReward)
	utils.DebugLog("target>>>>>>>>>>>>", target)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		type invite struct {
			CurrentCapacity uint64 `json:"currentCapacity"`
		}
		i := &invite{
			CurrentCapacity: target.CurrentCapacity,
		}
		utils.DebugLog("enter HTTPRewardFun write")
		write.Write(httpserv.NewJson(i, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPRewardFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPInviteFun HTTPInviteFun
func HTTPInviteFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspInvite)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		type invite struct {
			CapacityDelta   uint64 `json:"capacityDelta"`
			CurrentCapacity uint64 `json:"currentCapacity"`
		}
		i := &invite{
			CapacityDelta:   target.CapacityDelta,
			CurrentCapacity: target.CurrentCapacity,
		}
		data := make(map[string]*invite, 0)
		data["invite"] = i
		write.Write(httpserv.NewJson(data, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPInviteFun error ")
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

// HTTPSearchAlbumFun HTTPSearchAlbumFun
func HTTPSearchAlbumFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspSearchAlbum)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		list := make([]*myAlbum, 0)
		for _, f := range target.AlbumInfo {
			all := &myAlbum{
				AlbumBlurb: f.AlbumBlurb,
				AlbumId:    f.AlbumId,
				AlbumName:  f.AlbumName,
				AlbumTime:  f.AlbumTime,
				AlbumVisit: f.AlbumVisit,
				AlbumCover: f.AlbumCoverLink,
			}
			list = append(list, all)
		}
		data := make(map[string]interface{}, 0)
		data["albumList"] = list
		data["total"] = target.Total
		write.Write(httpserv.NewJson(data, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPSearchAlbumFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPAlbumContentFun HTTPAlbumContentFun
func HTTPAlbumContentFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	type albumContent struct {
		album    *myAlbum
		fileList []*allFile
	}
	target := httpRsp.Data.(*protos.RspAlbumContent)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		albumMap := make(map[string]*myAlbum, 0)
		album := &myAlbum{
			AlbumBlurb:    target.AlbumInfo.AlbumBlurb,
			AlbumId:       target.AlbumInfo.AlbumId,
			AlbumName:     target.AlbumInfo.AlbumName,
			AlbumTime:     target.AlbumInfo.AlbumTime,
			AlbumVisit:    target.AlbumInfo.AlbumVisit,
			AlbumCover:    target.AlbumInfo.AlbumCoverLink,
			IsPrivate:     target.AlbumInfo.IsPrivate,
			IsCollection:  target.IsCollection,
			WalletAddress: target.OwnerWalletAddress,
			AlbumType:     target.AlbumInfo.AlbumType,
		}
		albumMap["album"] = album
		list := make([]*allFile, 0)
		for _, f := range target.FileInfo {
			all := &allFile{
				FileHash:           f.FileHash,
				FileName:           f.FileName,
				FileSize:           f.FileSize,
				CreateTime:         f.CreateTime,
				IsDirectory:        f.IsDirectory,
				OwnerWalletAddress: f.OwnerWalletAddress,
				ID:                 f.SortId,
			}
			list = append(list, all)
		}
		albumInfo := make(map[string]interface{}, 0)
		albumInfo["album"] = albumMap["album"]
		albumInfo["fileList"] = list
		write.Write(httpserv.NewJson(albumInfo, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPAlbumContentFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPEditAlbumFun HTTPEditAlbumFun
func HTTPEditAlbumFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspEditAlbum)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		write.Write(httpserv.NewJson(nil, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPEditAlbumFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPFindMyAlbumFun HTTPFindMyAlbumFun
func HTTPFindMyAlbumFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspFindMyAlbum)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		list := make([]*myAlbum, 0)
		for _, f := range target.AlbumInfo {
			all := &myAlbum{
				AlbumId:    f.AlbumId,
				AlbumBlurb: f.AlbumBlurb,
				AlbumName:  f.AlbumName,
				AlbumVisit: f.AlbumVisit,
				AlbumTime:  f.AlbumTime,
				AlbumCover: f.AlbumCoverLink,
			}
			list = append(list, all)
		}

		data := make(map[string]interface{}, 0)
		data["albumList"] = list
		data["total"] = target.Total
		write.Write(httpserv.NewJson(data, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPFindMyAlbumFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPCreateAlbumFun HTTPCreateAlbumFun
func HTTPCreateAlbumFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspCreateAlbum)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		data := make(map[string]string, 0)
		data["albumId"] = target.AlbumId
		write.Write(httpserv.NewJson(data, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPCreateAlbumFun error ")
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

// HTTPDownPauseFun HTTPDownPauseFun
// func HTTPDownPauseFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
// 	target := httpRsp.Data.(*protos.RspDownloadSlicePause)
// 	if target.Result.State == protos.ResultState_RES_SUCCESS {
// 		write.Write(httpserv.NewJson(nil, setting.SUCCESSCode, target.Result.Msg).ToBytes())
// 		HTTPWriterMap.Delete(reqID)
// 		HTTPRspMap.Delete(reqID)
// 		return
// 	}
// 	utils.DebugLog("cuowu ")
// 	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
// 	HTTPWriterMap.Delete(reqID)
// 	HTTPRspMap.Delete(reqID)
// }

// HTTPGetConfigFun HTTPGetConfigFun
func HTTPGetConfigFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspConfig)
	type configData struct {
		IsUpgrade                   bool   `json:"IsUpgrade"`
		FreeCapacity                uint64 `json:"FreeCapacity"`
		Capacity                    uint64 `json:"Capacity"`
		DownloadPath                string `json:"DownloadPath"`
		IsCheckDefaultPath          bool   `json:"IsCheckDefaultPath"` // whether use default download path
		IsLimitDownloadSpeed        bool   `json:"IsLimitDownloadSpeed"`
		LimitDownloadSpeed          uint64 `json:"LimitDownloadSpeed"`
		IsLimitUploadSpeed          bool   `json:"IsLimitUploadSpeed"`
		LimitUploadSpeed            uint64 `json:"LimitUploadSpeed"`
		IsCheckFileOperation        bool   `json:"IsCheckFileOperation"`        // whether set file operation bubble
		IsCheckFileTransferFinished bool   `json:"IsCheckFileTransferFinished"` // whether set transfer finished sound alarm
		Invite                      uint64 `json:"Invite"`                      // how many invites
		InvitationCode              string `json:"InvitationCode"`              // invitation code
	}
	utils.DebugLog("target ", target)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		data := &configData{
			IsUpgrade:                   target.IsUpgrade,
			FreeCapacity:                target.FreeCapacity,
			Capacity:                    target.Capacity,
			DownloadPath:                setting.Config.DownloadPath,
			IsCheckDefaultPath:          setting.Config.IsCheckDefaultPath,
			IsLimitDownloadSpeed:        setting.Config.IsLimitDownloadSpeed,
			LimitDownloadSpeed:          setting.Config.LimitDownloadSpeed,
			IsLimitUploadSpeed:          setting.Config.IsLimitUploadSpeed,
			LimitUploadSpeed:            setting.Config.LimitUploadSpeed,
			IsCheckFileOperation:        setting.Config.IsCheckFileOperation,
			IsCheckFileTransferFinished: setting.Config.IsCheckFileTransferFinished,
			Invite:                      target.Invite,
			InvitationCode:              target.InvitationCode,
		}
		if strings.Contains(setting.Config.DownloadPath, "./") {
			dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
			if err != nil {
				utils.ErrorLog("dir err", err)
			}
			paths := strings.Split(setting.Config.DownloadPath, "./")
			data.DownloadPath = dir + "/" + paths[1]
			if setting.IsWindows {
				data.DownloadPath = filepath.FromSlash(data.DownloadPath)
			}
		}
		write.Write(httpserv.NewJson(data, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPGetConfigFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPMVdirFun HTTPMVdirFun
func HTTPMVdirFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspSaveFile)
	utils.DebugLog("httpRsp?>>>>>>>>>>>>")
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		write.Write(httpserv.NewJson(nil, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPMVdirFun error ")
	write.Write(httpserv.NewJson(nil, setting.FAILCode, target.Result.Msg).ToBytes())
	HTTPWriterMap.Delete(reqID)
	HTTPRspMap.Delete(reqID)
}

// HTTPMKdirFun HTTPMKdirFun
func HTTPMKdirFun(httpRsp *HTTPRsp, write http.ResponseWriter, reqID string) {
	target := httpRsp.Data.(*protos.RspMakeDirectory)
	if target.Result.State == protos.ResultState_RES_SUCCESS {
		write.Write(httpserv.NewJson(nil, setting.SUCCESSCode, target.Result.Msg).ToBytes())
		HTTPWriterMap.Delete(reqID)
		HTTPRspMap.Delete(reqID)
		return
	}
	utils.DebugLog("HTTPMKdirFun error ")
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
