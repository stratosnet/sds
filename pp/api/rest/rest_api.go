package rest

import (
	"context"

	"github.com/stratosnet/sds/pp/api"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"
)

func StartHTTPServ(ctx context.Context) {
	httpServ := httpserv.MyNewHTTPServ(setting.Config.Streaming.RestPort)
	httpServ.MyRoute("/videoSlice/", api.GetVideoSlice)
	httpServ.MyStart(ctx)
}
