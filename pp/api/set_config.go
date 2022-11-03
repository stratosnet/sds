package api

import (
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"
	"net/http"
)

func setConfig(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	key := ""
	value := ""
	if data["key"] != nil {
		key = data["key"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "key is required").ToBytes())
		return
	}

	if data["value"] != nil {
		value = data["value"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "value is required").ToBytes())
		return
	}

	if setting.SetConfig(key, value) {
		w.Write(httpserv.NewJson(nil, setting.SUCCESSCode, "change successfully").ToBytes())
		cf.SetLimitDownloadSpeed(setting.Config.LimitDownloadSpeed, setting.Config.IsLimitDownloadSpeed)
		cf.SetLimitUploadSpeed(setting.Config.LimitUploadSpeed, setting.Config.IsLimitUploadSpeed)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "change failed").ToBytes())
	}

}
