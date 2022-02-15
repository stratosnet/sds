package api

import (
	"net/http"
	"strings"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/httpserv"
)

func upProgress(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	if data["tasks"] != nil {
		type prog struct {
			TaskID   string  `json:"taskID"`
			Progress float32 `json:"progress"`
			Rate     int64   `json:"rate"`
			State    bool    `json:"state"`
		}
		ma := make(map[string]interface{}, 0)
		for _, f := range data["tasks"].([]interface{}) {
			gress := &prog{
				TaskID: f.(string),
				State:  false,
			}
			if val, ok := setting.UploadTaskIDMap.Load(f.(string)); ok {
				if p, ok := event.ProgressMap.Load(val.(string)); ok {
					pross := p.(float32)
					if pross > 100 {
						pross = 100
					}
					gress.Progress = pross
					gress.State = true
				}
				utils.DebugLog("gress.Progress", gress.Progress)
				utils.DebugLog("f>>>>>>>>>>>>>>>>>>>>", val.(string))
				gress.Rate = 0
				client.UpConnMap.Range(func(k, v interface{}) bool {
					if strings.HasPrefix(k.(string), val.(string)) {
						vconn := v.(*cf.ClientConn)
						w := vconn.GetSecondWriteFlow()
						gress.Rate += w
					}
					return true
				})
				ma[f.(string)] = gress
			}

		}
		m := make(map[string]interface{}, 0)
		m["taskList"] = ma
		w.Write(httpserv.NewJson(m, setting.SUCCESSCode, "request success").ToBytes())
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "fileHash is required").ToBytes())
	}
}
