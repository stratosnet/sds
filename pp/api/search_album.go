package api

import (
	"github.com/qsnetwork/sds/pp/event"
	"github.com/qsnetwork/sds/pp/setting"
	"github.com/qsnetwork/sds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func searchAlbum(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	keyword := ""
	albumType := "0"
	sortType := "1"
	var page uint64
	var number uint64
	if data["keyword"] != nil {
		keyword = data["keyword"].(string)
	}
	if data["albumType"] != nil {
		albumType = data["albumType"].(string)
	}
	if data["sortType"] != nil {
		sortType = data["sortType"].(string)
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
	event.SearchAlbum(keyword, albumType, sortType, uuid.New().String(), page, number, w)
}
