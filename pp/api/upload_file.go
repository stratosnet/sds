package api

import (
	"fmt"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/httpserv"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type upLoadFileResult struct {
	FilePath           string `json:"filePath"`
	State              bool   `json:"state"`
	TaskID             string `json:"taskID"`
	FailInfo           string `json:"failInfo"`
	FileName           string `json:"fileName"`
	FileSize           uint64 `json:"fileSize"`
	FileHash           string `json:"fileHash"`
	ImageWalletAddress string `json:"imageWalletAddress"`
}

func upLoadFile(w http.ResponseWriter, request *http.Request) {

	data, err := HTTPRequest(request, w, true)

	if err != nil {
		return
	}
	utils.DebugLog("path", data["walletAddress"])
	// sdPath := ""
	// if data["storagePath"] != nil {
	// 	sdPath = data["storagePath"].(string)
	// }
	type resData struct {
		path        string
		storagePath string
	}
	if data["tasks"] != nil {
		fileArr := data["tasks"].([]interface{})
		result := make(map[string][]*upLoadFileResult, 0)
		resultArr := make([]*upLoadFileResult, 0)
		if setting.CheckLogin() {
			for _, p := range fileArr {
				pathMap := p.(map[string]interface{})
				path := pathMap["path"].(string)
				sdPath := ""
				if pathMap["storagePath"] != nil {
					sdPath = pathMap["storagePath"].(string)
				}
				if setting.Iswindows {
					path = strings.Replace(path, `\`, "/", -1)
				}
				pathType := file.IsFile(path)
				if pathType == 0 {
					fmt.Println("wrong path")
					r := &upLoadFileResult{
						FilePath: path,
						State:    false,
						TaskID:   "",
						FailInfo: "wrong path",
						FileName: "",
						FileSize: 0,
					}
					resultArr = append(resultArr, r)
				} else if pathType == 1 {
					f := event.RequestUploadFileData(path, sdPath, "", false)
					go event.SendMessageToSPServer(f, header.ReqUploadFile)
					taskID := uuid.New().String()
					r := &upLoadFileResult{
						FailInfo: "",
						FilePath: path,
						State:    true,
						TaskID:   taskID,
						FileName: f.FileInfo.FileName,
						FileSize: f.FileInfo.FileSize,
					}
					setting.UpLoadTaskIDMap.Range(func(k, v interface{}) bool {
						if v.(string) == f.FileInfo.FileHash {
							r.TaskID = k.(string)
							return false
						}
						return true
					})
					setting.UpLoadTaskIDMap.Store(r.TaskID, f.FileInfo.FileHash)
					utils.DebugLog("taskid>>>>>>>>>>>>>>", r.TaskID)
					resultArr = append(resultArr, r)
				} else {
					utils.DebugLog("this is a file directory")
					file.GetAllFile(path)
					dir := filepath.Dir(path)
					for {
						select {
						case pathstring := <-setting.UpChan:
							utils.DebugLog("pathstring == ", pathstring)
							sPath := strings.Replace(pathstring, dir, "", -1)
							lastPaths := filepath.Dir(sPath)
							utils.DebugLog("lastPaths ==>>>>>>>>>>> ", lastPaths)
							if file.IsFile(pathstring) == 2 {
								event.MakeDirectory(sPath, uuid.New().String(), w)
							} else {
								var lps []string
								lps = strings.FieldsFunc(lastPaths, func(r rune) bool { return r == '/' })
								lastPaths = strings.Join(lps, "/")
								if sdPath != "" {
									lastPaths = sdPath + "/" + lastPaths
								}

								f := event.RequestUploadFileData(pathstring, lastPaths, "", false)
								utils.DebugLog("lastPaths>>>>", lastPaths)
								utils.DebugLog("storagePath+relativePath", lastPaths, pathstring)
								taskID := uuid.New().String()
								r := &upLoadFileResult{
									FailInfo: "",
									FilePath: path,
									State:    true,
									TaskID:   taskID,
									FileName: f.FileInfo.FileName,
									FileSize: f.FileInfo.FileSize,
								}
								setting.UpLoadTaskIDMap.Range(func(k, v interface{}) bool {
									if v.(string) == f.FileInfo.FileHash {
										r.TaskID = k.(string)
										return false
									}
									return true
								})
								setting.UpLoadTaskIDMap.Store(r.TaskID, f.FileInfo.FileHash)
								resultArr = append(resultArr, r)
								go event.SendMessageToSPServer(f, header.ReqUploadFile)
								utils.DebugLog("resust>>>>>>>>>>>>>>", resultArr)
							}

						default:
							result["list"] = resultArr
							w.Write(httpserv.NewJson(result, setting.SUCCESSCode, "request success").ToBytes())
							return
						}

					}
				}
			}
		} else {
			w.Write(httpserv.NewJson(nil, setting.FAILCode, "login first").ToBytes())
			return
		}
		result["list"] = resultArr
		w.Write(httpserv.NewJson(result, setting.SUCCESSCode, "request success").ToBytes())
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "path is required").ToBytes())
	}

}
