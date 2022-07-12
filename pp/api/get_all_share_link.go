package api

import (
	"net/http"
	"github.com/google/uuid"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
)

func getAllShareLink(w http.ResponseWriter, request *http.Request) {
	_, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	event.GetAllShareLink(uuid.New().String(), setting.WalletAddress, 0, w)
}
