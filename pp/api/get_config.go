package api

import (
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func getConfig(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	if data["walletAddress"] != nil && data["p2pAddress"] != nil {
		walletAddress := data["walletAddress"].(string)
		p2pAddress := data["p2pAddress"].(string)
		event.GetMyConfig(p2pAddress, walletAddress, uuid.New().String(), w)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "wallet address and P2P key address are required").ToBytes())
	}
}
