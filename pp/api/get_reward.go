package api

import (
	"github.com/qsnetwork/qsds/pp/event"
	"net/http"

	"github.com/google/uuid"
)

func getReward(w http.ResponseWriter, request *http.Request) {
	_, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	event.GetReward(uuid.New().String(), w)
}
