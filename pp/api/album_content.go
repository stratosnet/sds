package api

import (
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func albumContent(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	albumID := ""
	if data["albumID"] != nil {
		albumID = data["albumID"].(string)
		event.AlbumContent(albumID, uuid.New().String(), w)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "albumID is required").ToBytes())
	}
}
