package api

import (
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/pp/event"
	"github.com/qsnetwork/qsds/pp/setting"
	"github.com/qsnetwork/qsds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func fileSort(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	var files []*protos.FileInfo
	var albumID string
	if data["files"] != nil {
		for _, val := range data["files"].([]interface{}) {
			m := val.(map[string]interface{})
			t := &protos.FileInfo{
				FileHash: m["fileHash"].(string),
				SortId:   uint64(m["id"].(float64)),
			}
			files = append(files, t)
		}
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "files is required").ToBytes())
	}

	if data["albumID"] != nil {
		albumID = data["albumID"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "albumID is required").ToBytes())
	}
	event.FileSort(files, uuid.New().String(), albumID, w)
}
