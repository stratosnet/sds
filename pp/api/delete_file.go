package api

import (
	"github.com/qsnetwork/qsds/pp/event"
	"github.com/qsnetwork/qsds/pp/setting"
	"github.com/qsnetwork/qsds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func deleteFile(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	fileHash := ""
	path := ""
	if data["fileHash"] != nil {
		fileHash = data["fileHash"].(string)
		event.DeleteFile(fileHash, uuid.New().String(), w)
	}

	if data["path"] != nil {
		path = data["path"].(string)
		event.RemoveDirectory(path, uuid.New().String(), w)
	}
	if data["fileHash"] == nil && data["path"] == nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "either fileHash/path is required").ToBytes())
	}
}
