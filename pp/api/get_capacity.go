package api

import (
	"github.com/stratosnet/sds/pp/event"
	"net/http"

	"github.com/google/uuid"
)

func getCapacity(w http.ResponseWriter, request *http.Request) {
	_, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	event.GetCapacity(uuid.New().String(), w)

}
