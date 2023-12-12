package rest

import (
	"context"

	"github.com/stratosnet/sds/pp/api"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"
)

func StartHTTPServ(ctx context.Context) {
	httpServ := httpserv.MyNewHTTPServ(setting.Config.Streaming.RestPort)
	httpServ.MyRoute("/prepareVideoFileCache/", api.PrepareVideoFileCache) // download to cache
	httpServ.MyRoute("/getVideoSliceCache/", api.GetVideoSliceCache)       // get from cache
	httpServ.MyRoute("/findVideoSlice/", api.GetVideoSlice)                // redirect
	httpServ.MyRoute("/videoSlice/", api.GetVideoSlice)
	httpServ.MyStart(ctx)
}
