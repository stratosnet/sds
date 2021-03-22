package api

import (
	"github.com/qsnetwork/qsds/pp/event"
	"github.com/qsnetwork/qsds/pp/setting"
	"github.com/qsnetwork/qsds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func moveFileDirectory(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	fileHash := ""
	originalDir := ""
	targetDir := ""
	if data["fileHash"] != nil {
		fileHash = data["fileHash"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "fileHash is required").ToBytes())
		return
	}

	if data["originalDir"] != nil {
		originalDir = data["originalDir"].(string)
	}

	if data["targetDir"] != nil {
		targetDir = data["targetDir"].(string)
	}

	if originalDir == targetDir {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "destination directory must be different to original").ToBytes())
		return
	}
	event.MoveFileDirectory(fileHash, originalDir, targetDir, uuid.New().String(), w)
}
