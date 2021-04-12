package api

import (
	"github.com/stratosnet/sds/pp/event"
	"net/http"

	"github.com/google/uuid"
)

func getAllShareLink(w http.ResponseWriter, request *http.Request) {
	_, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	event.GetAllShareLink(uuid.New().String(), w)
}
