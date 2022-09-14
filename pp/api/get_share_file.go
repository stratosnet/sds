package api

import (
	"context"
	"net/http"

	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"

	"github.com/google/uuid"
)

func getShareFile(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	keyword := ""
	sharePassword := ""
	saveAs := ""
	if data["keyword"] != nil {
		keyword = data["keyword"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "keyword is required").ToBytes())
		return
	}

	if data["sharePassword"] != nil {
		sharePassword = data["sharePassword"].(string)
	}

	if data["saveAs"] != nil {
		saveAs = data["saveAs"].(string)
	}

	event.GetShareFile(context.Background(), keyword, sharePassword, saveAs, uuid.New().String(), setting.WalletAddress, setting.WalletPublicKey, nil, w)
}
