package api

import (
	"net/http"

	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"

	"github.com/google/uuid"
)

func shareFile(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}

	fileHash := ""
	pathHash := ""
	isDirectory := false
	isPrivate := false
	var shareTime int64

	if data["isDirectory"] != nil {
		isDirectory = data["isDirectory"].(bool)
	}

	if data["fileHash"] != nil {
		if isDirectory {
			pathHash = data["fileHash"].(string)
		} else {
			fileHash = data["fileHash"].(string)
		}
	}

	if data["isPrivate"] != nil {
		isPrivate = data["isPrivate"].(bool)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "isPrivate is required").ToBytes())
		return
	}

	if data["shareTime"] != nil {
		shareTime = int64(data["shareTime"].(float64))
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "shareTime is required").ToBytes())
		return
	}

	event.GetReqShareFile(uuid.New().String(), fileHash, pathHash, shareTime, isPrivate, w)
}
