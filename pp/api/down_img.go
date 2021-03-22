package api

import (
	"github.com/qsnetwork/qsds/pp/event"
	"github.com/qsnetwork/qsds/pp/setting"
	"github.com/qsnetwork/qsds/utils"
	"github.com/qsnetwork/qsds/utils/httpserv"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
)

func downImg(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	if data["path"] != nil {
		path := data["path"].(string)
		ps := strings.Split(path, "/")
		fileHash := ps[len(ps)-1]
		setting.ImageMap.Store(fileHash, fileHash)
		var f *os.File
		var err error
		openPath := setting.IMAGEPATH + fileHash
		// openPath = filepath.FromSlash(openPath)
		utils.DebugLog("openpath>>>>>>", openPath)
		f, err = os.Open(openPath)
		if err != nil {
			event.GetFileStorageInfo(path, "images", uuid.New().String(), true, w)
		} else {
			data1 := make(map[string]interface{}, 0)
			img, err := ioutil.ReadAll(f)
			if err != nil {
				data1["image"] = ""
			}
			data1["image"] = img
			w.Write(httpserv.NewJson(data1, setting.SUCCESSCode, "request success").ToBytes())
		}

	}
}
