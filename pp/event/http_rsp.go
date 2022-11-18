package event

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/stratosnet/sds/framework/core"
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
	HTTPDownloadSlice HTTPType = 1 + iota
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

func putData(ctx context.Context, httpType HTTPType, target interface{}) {
	reqId := core.GetRemoteReqId(ctx)
	if reqId == "" {
		return
	}

	if setting.Config.InternalPort != "" && setting.WalletAddress != "" {
		// httpRsp(target.ReqId, HTTPDownloadFile, &target)
		rsp := &HTTPRsp{
			Data: target,
			Type: httpType,
		}
		HTTPRspMap.Store(reqId, rsp)
	}
}

func storeResponseWriter(ctx context.Context, w http.ResponseWriter) error {
	if setting.Config.InternalPort != "" && setting.WalletAddress != "" {
		if w != nil {
			return StoreReqID(ctx, w)
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
		case HTTPDownloadSlice:
			{
				HTTPDownloadSliceFun(httpRsp, write, reqID)
				return nil
			}
		}
	}
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

// StoreReqID
func StoreReqID(ctx context.Context, w http.ResponseWriter) error {
	reqId := core.GetRemoteReqId(ctx)
	if reqId == "" {
		return nil
	}
	HTTPWriterMap.Store(reqId, w)
	return HTTPStartListen(reqId)
}
