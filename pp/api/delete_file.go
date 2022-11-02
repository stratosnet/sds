package api

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"
)

func deleteFile(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	fileHash := ""
	//path := ""
	if data["fileHash"] != nil {
		fileHash = data["fileHash"].(string)
		ctx := core.RegisterRemoteReqId(context.Background(), uuid.New().String())
		event.DeleteFile(ctx, fileHash, w)
	}

	//if data["path"] != nil {
	//	path = data["path"].(string)
	//	event.RemoveDirectory(path, uuid.New().String(), w)
	//}
	if data["fileHash"] == nil && data["path"] == nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "either fileHash/path is required").ToBytes())
	}
}
