package api

import (
	"context"

	"github.com/stratosnet/sds/framework/utils/httpserv"
	"github.com/stratosnet/sds/pp/setting"
)

func StartHTTPServ(ctx context.Context) {
	httpServ := httpserv.MyNewHTTPServ(setting.Config.Streaming.InternalPort)
	httpServ.MyRoute("/streamVideoStorageInfo/", streamVideoInfoCache)
	httpServ.MyRoute("/streamSharedVideoStorageInfo/", streamSharedVideoInfoCache)
	httpServ.MyRoute("/streamVideo/", streamVideoP2P)
	httpServ.MyRoute("/streamVideoStorageInfoHttp/", streamVideoInfoHttp)
	httpServ.MyRoute("/streamVideoHttp/", streamVideoHttp)
	httpServ.MyRoute("/clearStreamTask/", clearStreamTask)
	httpServ.MyStart(ctx)
}
