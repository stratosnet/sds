package api

import (
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func deleteAlbum(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	if data["albumID"] != nil {
		albumID := data["albumID"].(string)
		event.DeleteAlbum(albumID, uuid.New().String(), w)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "albumID is required").ToBytes())
	}

}
