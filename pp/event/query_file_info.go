package event

// Author j
import (
	"context"
	"fmt"
	"net/http"

	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/httpserv"
)

var isImage bool

// DirectoryTreeMap DirectoryTreeMap
var DirectoryTreeMap = make(map[string]*Ts)

// Ts Ts
type Ts struct {
	Reqs  []*protos.RspFindDirectoryTree
	Count int
}

// Tree Tree
// type Tree struct {
// 	Dir  string
// 	Hash string
// }

// DirectoryTree
// var DirectoryTree = make(map[string]*Tree, 10)

// whether is found from query
var isFind bool

// FindDirectoryTree
func FindDirectoryTree(reqID, pathHash string, w http.ResponseWriter, isF bool) {
	if setting.CheckLogin() {
		// request is the same as AlbumContent
		sendMessage(client.PPConn, reqFindDirectoryTreeData(reqID, pathHash), header.ReqFindDirectoryTree)
		stroeResponseWriter(reqID, w)
		isFind = isF
	} else {
		notLogin(w)
	}
}

// ReqFindDirectoryTree
func ReqFindDirectoryTree(ctx context.Context, conn spbf.WriteCloser) {
	transferSendMessageToSPServer(spbf.MessageFromContext(ctx))
}

// RspFindDirectoryTree
func RspFindDirectoryTree(ctx context.Context, conn spbf.WriteCloser) {
	utils.DebugLog("target>>context>>>>>>>>>>>>>>>>>>>")
	var target protos.RspFindDirectoryTree
	if unmarshalData(ctx, &target) {
		if isFind {
			putData(target.ReqId, HTTPDirectoryTree, &target)
		}
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				utils.DebugLog("target>>>>>>>>>>>>>>>>>>>>>", target)
				ts := DirectoryTreeMap[target.ReqId]
				ts.Reqs = append(ts.Reqs, &target)
				utils.DebugLog("Reqs>>>>>>>>>>>>>>>>>>>>>", len(ts.Reqs))
			} else {
				fmt.Println("action  failed", target.Result.Msg)
			}
		} else {
			transferSendMessageToClient(target.P2PAddress, spbf.MessageFromContext(ctx))
		}
	}
}

// GetFileStorageInfo p to pp
func GetFileStorageInfo(path, savePath, reqID string, isImg bool, w http.ResponseWriter) {
	if setting.CheckLogin() {
		if CheckDownloadPath(path) {
			utils.DebugLog("path:", path)
			sendMessage(client.PPConn, reqFileStorageInfoData(path, savePath, reqID), header.ReqFileStorageInfo)
			if isImg {
				isImage = isImg
				stroeResponseWriter(reqID, w)
			}
		} else {
			utils.ErrorLog("please input correct download link, eg: spb://address/fileHash|filename(optional)")
			if w != nil {
				w.Write(httpserv.NewJson(nil, setting.FAILCode, "please input correct download link, eg:  spb://address/fileHash|filename(optional)").ToBytes())
			}
		}
	} else {
		notLogin(w)
	}
}

// ReqFileStorageInfo  P-PP , PP-SP
func ReqFileStorageInfo(ctx context.Context, conn spbf.WriteCloser) {
	utils.Log("pp get ReqFileStorageInfo directly transfer to SP")
	transferSendMessageToSPServer(spbf.MessageFromContext(ctx))
}

// RspFileStorageInfo SP-PP , PP-P
func RspFileStorageInfo(ctx context.Context, conn spbf.WriteCloser) {
	// PP check whether itself is the storage PP, if not transfer
	utils.Log("get，RspFileStorageInfo")
	var target protos.RspFileStorageInfo
	if unmarshalData(ctx, &target) {

		utils.DebugLog("file hash", target.FileHash)
		// utils.Log("target", target.P2PAddress)
		if target.P2PAddress == setting.P2PAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				fmt.Println("download starts: ")
				task.DownloadFileMap.Store(target.FileHash, &target)
				DownloadFileSlice(&target)
				utils.DebugLog("DownloadFileSlice(&target)", target)
			} else {
				fmt.Println("failed to download，", target.Result.Msg)
			}
			if isImage {
				putData(target.ReqId, HTTPDownloadFile, &target)
			}
		} else {
			// store the task and transfer
			task.AddDownloadTask(&target)
			transferSendMessageToClient(target.P2PAddress, rspFileStorageInfoData(&target))
		}
	}
}

// CheckDownloadPath
func CheckDownloadPath(path string) bool {

	if len(path) < setting.Config.DownloadPathMinLen {
		utils.DebugLog("invalid path length")
		return false
	}
	if path[:6] != "spb://" {
		return false
	}
	if path[47:48] != "/" {
		return false
	}
	return true
}
