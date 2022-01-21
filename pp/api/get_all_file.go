package api

import (
	"net/http"

	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"

	"github.com/google/uuid"
)

// case body.Directory == "" AND body.fileHash == "": query all files under root
// case body.Directory != "" AND body.fileHash == "": query all files under the directory specified
// case body.Directory == "" AND body.fileHash != "": query the file specified under root
// case body.Directory != "" AND body.fileHash != "": query the file specified under the directory specified

func getAllFile(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	var directory string
	var fileName string
	fileType := 0
	isUp := true
	var keyword string
	if data["directory"] != nil {
		directory = data["directory"].(string)
	} else {
		directory = ""
	}

	if data["fileName"] != nil {
		fileName = data["fileName"].(string)
	} else {
		fileName = ""
	}
	if data["fileType"] != nil {
		fileType = int(data["fileType"].(float64))
	}

	if data["isUp"] != nil {
		isUp = data["isUp"].(bool)
	}

	if data["keyword"] != nil {
		keyword = data["keyword"].(string)
	} else {
		keyword = ""
	}
	if setting.CheckLogin() {
		event.FindMyFileList(fileName, directory, uuid.New().String(), keyword, fileType, isUp, w)
	}
}
