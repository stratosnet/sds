package api

import (
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func uploadCoverImage(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	if data["path"] != nil {
		path := data["path"].(string)
		// tmpString, err := utils.ImageCommpress(path)
		// if err) {
		// 	utils.ErrorLog("imagepath>>>>", err)
		// 	w.Write(httpserv.NewJson(nil, setting.FAILCode, "compression failed").ToBytes())
		// 	return
		// }
		// f := event.RequestUploadFileData(tmpString, "", true)
		// event.SendMessageToSPServer(f, header.ReqUploadFile)
		event.RequestUploadCoverImage(path, uuid.New().String(), w)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "path is required").ToBytes())
	}

}
