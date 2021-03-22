package api

import (
	"github.com/qsnetwork/qsds/pp/event"
	"github.com/qsnetwork/qsds/pp/setting"
	"github.com/qsnetwork/qsds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func saveFolder(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	folderHash := ""
	ownerAddress := ""
	if data["folderHash"] != nil {
		folderHash = data["folderHash"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "folderHash is required").ToBytes())
		return
	}
	if data["ownerAddress"] != nil {
		ownerAddress = data["ownerAddress"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "ownerAddress is required").ToBytes())
		return
	}
	event.SaveFolder(folderHash, ownerAddress, uuid.New().String(), w)
}
