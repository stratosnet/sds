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

func deleteShare(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	shareID := ""
	if data["shareID"] != nil {
		shareID = data["shareID"].(string)
		ctx := core.RegisterRemoteReqId(context.Background(), uuid.New().String())
		event.DeleteShare(ctx, shareID, setting.WalletAddress, w)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "shareID is required").ToBytes())
	}
}
