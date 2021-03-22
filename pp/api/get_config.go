package api

import (
	"github.com/qsnetwork/qsds/pp/event"
	"github.com/qsnetwork/qsds/pp/setting"
	"github.com/qsnetwork/qsds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func getConfig(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	if data["walletAddress"] != nil {
		walletAddress := data["walletAddress"].(string)
		event.GetMyConfig(walletAddress, uuid.New().String(), w)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "wallet address is required").ToBytes())
	}
}
