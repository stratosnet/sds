package api

import (
	"github.com/qsnetwork/sds/pp/event"
	"github.com/qsnetwork/sds/pp/setting"
	"github.com/qsnetwork/sds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func deleteShare(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	shareID := ""
	if data["shareID"] != nil {
		shareID = data["shareID"].(string)
		event.DeleteShare(shareID, uuid.New().String(), w)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "shareID is required").ToBytes())
	}
}
