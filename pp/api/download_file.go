package api

import (
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/pp/event"
	"github.com/qsnetwork/sds/pp/setting"
	"github.com/qsnetwork/sds/utils"
	"github.com/qsnetwork/sds/utils/httpserv"
	"net/http"
	"os"

	"github.com/google/uuid"
)

type downFile struct {
	isDirectory     bool   `json:"isDirectory"`
	hash            string `json:"hash"`
	belongToAddress string `json:"belongToAddress"`
}

func downloadFile(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}

	// if data["belongToAddress"] == nil {
	// 	w.Write(httpserv.NewJson(nil, setting.FAILCode, "belongToAddress is required").ToBytes())
	// 	return
	// }

	if data["savePath"] != nil {
		setting.Config.DownloadPath = data["savePath"].(string)
	}
	type df struct {
		TaskID   string `json:"taskID"`
		Path     string `json:"path"`
		FileName string `json:"fileName"`
		FileSize uint64 `json:"fileSize"`
	}
	list := make([]*df, 0)
	isDirectory := false
	if data["path"] != nil {
		reqID := uuid.New().String()

		count := 0
		fileArr := data["path"].([]interface{})
		for _, f := range fileArr {
			m := f.(map[string]interface{})
			if m["hash"] == nil {
				w.Write(httpserv.NewJson(nil, setting.FAILCode, "hash is required").ToBytes())
				return
			}
			if m["isDirectory"] != nil {
				isDirectory = m["isDirectory"].(bool)
			}
			if m["belongToAddress"] == nil {
				w.Write(httpserv.NewJson(nil, setting.FAILCode, "belongToAddress is required").ToBytes())
				return
			}
			p := &downFile{
				hash:            m["hash"].(string),
				isDirectory:     isDirectory,
				belongToAddress: m["belongToAddress"].(string),
			}
			if p.isDirectory {
				go event.FindDirectoryTree(reqID, p.hash, w, false)
				count++
			} else {
				path := "spb://" + p.belongToAddress + "/" + p.hash
				downTaskID := uuid.New().String()
				tree := &df{
					TaskID: downTaskID,
					Path:   "",
				}
				setting.DownLoadTaskIDMap.Range(func(k, v interface{}) bool {
					if v.(string) == p.hash {
						tree.TaskID = k.(string)
						return false
					}
					return true
				})
				setting.DownLoadTaskIDMap.Store(tree.TaskID, p.hash)
				list = append(list, tree)
				event.GetFileStorageInfo(path, "", uuid.New().String(), false, w)
			}
		}

		event.DirectoryTreeMap[reqID] = &event.Ts{
			Reqs:  make([]*protos.RspFindDirectoryTree, 0),
			Count: count,
		}
		utils.DebugLog("88888>>>>>>>>>>>!!!!!!!!!!!!!!!!!!!", event.DirectoryTreeMap[reqID].Count)

		for {
			if event.DirectoryTreeMap[reqID].Count == len(event.DirectoryTreeMap[reqID].Reqs) {

				reqs := event.DirectoryTreeMap[reqID]
				for _, target := range reqs.Reqs {
					for _, finfo := range target.FileInfo {
						if !finfo.IsDirectory {
							utils.DebugLog("file is at 》》》》》》》", finfo.StoragePath)
							downTaskID := uuid.New().String()
							tree := &df{
								TaskID:   downTaskID,
								Path:     finfo.StoragePath,
								FileName: finfo.FileName,
								FileSize: finfo.FileSize,
							}
							path := "spb://" + finfo.OwnerWalletAddress + "/" + tree.TaskID
							setting.DownLoadTaskIDMap.Range(func(k, v interface{}) bool {
								if v.(string) == finfo.FileHash {
									tree.TaskID = k.(string)
									return false
								}
								return true
							})
							setting.DownLoadTaskIDMap.Store(tree.TaskID, finfo.FileHash)
							event.GetFileStorageInfo(path, finfo.StoragePath, uuid.New().String(), false, w)
							list = append(list, tree)
						} else {
							utils.DebugLog("directory is at 》》》》》》》", finfo.StoragePath)
							if utils.CheckError(os.MkdirAll(setting.Config.DownloadPath+finfo.StoragePath+"/"+finfo.FileName, os.ModePerm)) {
								utils.ErrorLog("err>>>>>>>::::::", err)
							}
						}
					}
				}
				result := make(map[string][]*df, 0)
				result["list"] = list
				w.Write(httpserv.NewJson(result, setting.SUCCESSCode, "request success").ToBytes())
				return
			}
		}

		// for {
		// 	if
		// 	select {
		// 	case dt := <-event.DirectoryTree:
		// 		tree := &df{
		// 			FileHash: dt.Hash,
		// 			Path:     dt.Dir,
		// 		}
		// 		// path := "spb://" + data["belongAddress"].(string) + "/" + tree.FileHash
		// 		// event.GetFileStorageInfo(path, savePath, uuid.New().String(), w)
		// 		list = append(list, tree)
		// 		utils.DebugLog("DirectoryTree>>>>>>>>>>>")
		// 	default:
		// 		utils.DebugLog("DirectoryTree.............")
		// 		result := make(map[string][]*df, 0)
		// 		result["list"] = list
		// 		w.Write(httpserv.NewJson(result, setting.SUCCESSCode, "request success").ToBytes())
		// 		return
		// 	}

		// }
	}

	// if data["directory"] != nil {
	// 	fileArr := data["directory"].([]interface{})
	// 	for _, f := range fileArr {
	// 		p := f.(string)
	// 		event.FindDirectoryTree(uuid.New().String(), p, w)
	// 	}
	// 	result := make(map[string][]*down, 0)
	// 	result["list"] = list
	// 	w.Write(httpserv.NewJson(result, setting.SUCCESSCode, "request success").ToBytes())
	// }

}
