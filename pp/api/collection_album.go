package api

import (
	"github.com/qsnetwork/sds/pp/event"
	"github.com/qsnetwork/sds/pp/setting"
	"github.com/qsnetwork/sds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func collectionAlbum(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	isCollection := false
	if data["isCollection"] != nil {
		isCollection = data["isCollection"].(bool)
	}
	if data["albumID"] != nil {
		albumID := data["albumID"].(string)
		event.CollectionAlbum(albumID, uuid.New().String(), isCollection, w)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "albumID is required").ToBytes())
		return
	}

}
