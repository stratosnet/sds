package rest

import (
	"context"

	"github.com/stratosnet/sds/pp/api"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"
)

func StartHTTPServ(ctx context.Context) {
	httpServ := httpserv.MyNewHTTPServ(setting.Config.Streaming.RestPort)
	httpServ.MyRoute("/getOzone/", api.GetOzone)
	httpServ.MyRoute("/prepareVideoFileCache/", api.PrepareVideoFileCache)
	httpServ.MyRoute("/getVideoSliceCache/", api.GetVideoSliceCache)
	httpServ.MyRoute("/findVideoSlice/", api.GetVideoSlice)
	httpServ.MyStart(ctx)
}
