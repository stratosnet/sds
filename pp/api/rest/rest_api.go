package rest

import (
	"github.com/stratosnet/sds/pp/api"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"
)

func StartHTTPServ() {
	httpServ := httpserv.MyNewHTTPServ(setting.Config.RestPort)
	httpServ.MyRoute("/videoSlice/", api.GetVideoSlice)
	httpServ.MyStart()
}
