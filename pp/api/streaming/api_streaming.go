package streaming

import (
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"
)

func StartHTTPServ() {
	httpServ := httpserv.MyNewHTTPServ(setting.Config.StreamingPort)
	httpServ.MyRoute("/streamVideo/", streamVideo)
	httpServ.MyRoute("/videoSlice/", getVideoSlice)
	httpServ.MyRoute("/fileStorageInfo/", fileStorageInfo)
	httpServ.MyStart()
}
