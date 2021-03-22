package api

import (
	"github.com/qsnetwork/qsds/pp/event"
	"github.com/qsnetwork/qsds/pp/setting"
	"github.com/qsnetwork/qsds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func myCollectionAlbum(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	albumType := "0"
	keyword := ""
	var page uint64
	var number uint64
	if data["albumType"] != nil {
		albumType = data["albumType"].(string)
	}

	if data["keyword"] != nil {
		keyword = data["keyword"].(string)
	}

	if data["page"] != nil {
		page = uint64(data["page"].(float64))
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "page is required").ToBytes())
		return
	}
	if data["number"] != nil {
		number = uint64(data["number"].(float64))
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "number is required").ToBytes())
		return
	}
	event.MyCollectionAlbum(albumType, uuid.New().String(), page, number, keyword, w)
}
