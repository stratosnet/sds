package api

import (
	"github.com/stratosnet/sds/pp/event"
	"net/http"

	"github.com/google/uuid"
)

func directory(w http.ResponseWriter, request *http.Request) {
	_, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	event.FindDirectory(uuid.New().String(), w)
}
