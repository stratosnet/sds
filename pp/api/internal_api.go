package api

import (
	"context"

	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"
)

func StartHTTPServ(ctx context.Context) {
	httpServ := httpserv.MyNewHTTPServ(setting.Config.InternalPort)
	httpServ.MyRoute("/streamVideoStorageInfo/", streamVideoInfoCache)
	httpServ.MyRoute("/streamSharedVideoStorageInfo/", streamSharedVideoInfoCache)
	httpServ.MyRoute("/streamVideo/", streamVideoP2P)
	httpServ.MyRoute("/streamVideoStorageInfoHttp/", streamVideoInfoHttp)
	httpServ.MyRoute("/streamVideoHttp/", streamVideoHttp)
	httpServ.MyRoute("/clearStreamTask/", clearStreamTask)
	httpServ.MyStart(ctx)
}
