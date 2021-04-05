package api

import (
	"github.com/qsnetwork/sds/pp/event"
	"net/http"

	"github.com/google/uuid"
)

func findMyAlbum(w http.ResponseWriter, request *http.Request) {
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
	}
	if data["number"] != nil {
		number = uint64(data["number"].(float64))
	}
	event.FindMyAlbum(uuid.New().String(), page, number, albumType, keyword, w)
}
