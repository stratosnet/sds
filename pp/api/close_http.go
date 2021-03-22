package api

import (
	"net/http"
	"os"
)

func closeHTTP(w http.ResponseWriter, request *http.Request) {
	os.Exit(-1)
}
