package api

import (
	"github.com/qsnetwork/sds/pp/event"
	"github.com/qsnetwork/sds/pp/setting"
	"github.com/qsnetwork/sds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func invite(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	if data["code"] != nil {
		code := data["code"].(string)
		event.Invite(code, uuid.New().String(), w)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "code is required").ToBytes())
		return
	}

}
