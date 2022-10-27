package api

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
)

func getAllShareLink(w http.ResponseWriter, request *http.Request) {
	_, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	ctx := core.RegisterRemoteReqId(context.Background(), uuid.New().String())
	event.GetAllShareLink(ctx, setting.WalletAddress, 0, w)
}
