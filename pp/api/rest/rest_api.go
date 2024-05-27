package rest

import (
	"context"
	"net/http"

	"github.com/stratosnet/sds/framework/utils/httpserv"
	"github.com/stratosnet/sds/pp/api"
	"github.com/stratosnet/sds/pp/setting"
)

func StartHTTPServ(ctx context.Context) {
	httpServ := httpserv.MyNewHTTPServ(setting.Config.Streaming.RestPort)
	httpServ.MyRoute("/getOzone/", corsHandler(api.GetOzone))
	httpServ.MyRoute("/prepareVideoFileCache/", corsHandler(api.PrepareVideoFileCache))
	httpServ.MyRoute("/prepareSharedVideoFileCache/", corsHandler(api.PrepareSharedVideoFileCache))
	httpServ.MyRoute("/getVideoSliceCache/", corsHandler(api.GetVideoSliceCache))
	httpServ.MyRoute("/findVideoSlice/", corsHandler(api.GetVideoSlice))
	httpServ.MyStart(ctx)
}

func corsHandler(h func(w http.ResponseWriter, req *http.Request)) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		} else {
			h(w, r)
		}
	}
}
