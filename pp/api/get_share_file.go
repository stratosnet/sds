package api

import (
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func getShareFile(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	keyword := ""
	sharePassword := ""
	if data["keyword"] != nil {
		keyword = data["keyword"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "keyword is required").ToBytes())
		return
	}

	if data["sharePassword"] != nil {
		sharePassword = data["sharePassword"].(string)
	}
	event.GetShareFile(keyword, sharePassword, uuid.New().String(), w)
}
