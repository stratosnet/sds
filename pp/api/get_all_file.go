package api

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
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
	var pageId uint64
	var fileName string
	fileType := 0
	isUp := true
	var keyword string

	if data["pageId"] != nil {
		pageId = data["pageId"].(uint64)
	} else {
		pageId = 0
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
		ctx := core.RegisterRemoteReqId(context.Background(), uuid.New().String())
		event.FindFileList(ctx, fileName, setting.WalletAddress, pageId, keyword, fileType, isUp, w)
	}
}
