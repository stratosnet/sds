package api

import (
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func saveFile(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	fileHash := ""
	ownerAddress := ""
	if data["fileHash"] != nil {
		fileHash = data["fileHash"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "fileHash is required").ToBytes())
		return
	}
	if data["ownerAddress"] != nil {
		ownerAddress = data["ownerAddress"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "ownerAddress is required").ToBytes())
		return
	}

	event.SaveOthersFile(fileHash, ownerAddress, uuid.New().String(), w)
}
