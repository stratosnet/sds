package api

import (
	"context"
	"net/http"

	"github.com/stratosnet/sds/framework/utils/httpserv"
	"github.com/stratosnet/sds/pp/setting"
)

func StartHTTPServ(ctx context.Context) {
	httpServ := httpserv.MyNewHTTPServ(setting.Config.Streaming.InternalPort)
	httpServ.MyRoute("/streamVideoStorageInfo/", corsHandler(streamVideoInfoCache))
	httpServ.MyRoute("/streamSharedVideoStorageInfo/", corsHandler(streamSharedVideoInfoCache))
	httpServ.MyRoute("/streamVideo/", corsHandler(streamVideoP2P))
	httpServ.MyRoute("/streamVideoStorageInfoHttp/", streamVideoInfoHttp)
	httpServ.MyRoute("/streamVideoHttp/", streamVideoHttp)
	httpServ.MyRoute("/clearStreamTask/", clearStreamTask)
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
