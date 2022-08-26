package api

import (
	"context"
	"net/http"

	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"

	"github.com/google/uuid"
)

func deleteShare(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	shareID := ""
	if data["shareID"] != nil {
		shareID = data["shareID"].(string)
		event.DeleteShare(context.Background(), shareID, uuid.New().String(), setting.WalletAddress, w)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "shareID is required").ToBytes())
	}
}
