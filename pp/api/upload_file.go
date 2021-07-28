package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/httpserv"

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

	// check differently for sp
	if !setting.CheckLogin() {
		_, _ = w.Write(httpserv.NewJson(nil, setting.FAILCode, "login first").ToBytes())
		return
	}

	data, err := HTTPRequest(request, w, true)

	if err != nil {
		return
	}
	utils.DebugLog("path", data["walletAddress"])

	if data["tasks"] == nil {
		_, _ = w.Write(httpserv.NewJson(nil, setting.FAILCode, "path is required").ToBytes())
		return
	}

	fileArr := data["tasks"].([]interface{})
	result := make(map[string][]*upLoadFileResult, 0)
	resultArr := make([]*upLoadFileResult, 0)

	for _, p := range fileArr {
		pathMap := p.(map[string]interface{})
		path := pathMap["path"].(string)

		sdPath := ""
		if pathMap["storagePath"] != nil {
			sdPath = pathMap["storagePath"].(string)
		}
		if setting.IsWindows {
			path = strings.Replace(path, `\`, "/", -1)
		}
		isFile := false
		isFile, err = file.IsFile(path)

		if err != nil {
			fmt.Println(err)
			r := &upLoadFileResult{
				FilePath: path,
				State:    false,
				TaskID:   "",
				FailInfo: "wrong path",
				FileName: "",
				FileSize: 0,
			}
			resultArr = append(resultArr, r)
			continue
		}

		if isFile {
			f := event.RequestUploadFileData(path, sdPath, "", false, false)
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
			utils.DebugLogf("Upload task ID >> %v", r.TaskID)
			resultArr = append(resultArr, r)
			continue
		}

		// ----------------------------------------
		// is directory
		utils.DebugLog("this is a file directory")

		/*

			TODO change this func to not directly send file into chan, because total file size needs to be calculated first

		*/
		file.GetAllFiles(path) // this func should be per connection for sp

		dir := filepath.Dir(path)
		for {
			select {
			case pathstring := <-setting.UpChan:
				utils.DebugLog("pathstring == ", pathstring)
				sPath := strings.Replace(pathstring, dir, "", -1)
				lastPaths := filepath.Dir(sPath)
				utils.DebugLog("lastPaths ==>>>>>>>>>>> ", lastPaths)
				if isFile, _ = file.IsFile(pathstring); !isFile {
					event.MakeDirectory(sPath, uuid.New().String(), w)
					continue
				}

				var lps []string
				lps = strings.FieldsFunc(lastPaths, func(r rune) bool { return r == '/' })
				lastPaths = strings.Join(lps, "/")
				if sdPath != "" {
					lastPaths = sdPath + "/" + lastPaths
				}

				f := event.RequestUploadFileData(pathstring, lastPaths, "", false, false)
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

			default:
				result["list"] = resultArr
				w.Write(httpserv.NewJson(result, setting.SUCCESSCode, "request success").ToBytes())
				return
			}

		}
	}
	result["list"] = resultArr
	_, _ = w.Write(httpserv.NewJson(result, setting.SUCCESSCode, "request success").ToBytes())
}
