package api

import (
	"github.com/qsnetwork/sds/framework/client/cf"
	"github.com/qsnetwork/sds/pp/client"
	"github.com/qsnetwork/sds/pp/event"
	"github.com/qsnetwork/sds/pp/setting"
	"github.com/qsnetwork/sds/utils"
	"github.com/qsnetwork/sds/utils/httpserv"
	"net/http"
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
			if val, ok := setting.UpLoadTaskIDMap.Load(f.(string)); ok {
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
				if up, ok := client.UpConnMap.Load(val.(string)); ok {
					vconn := up.(*cf.ClientConn)
					w := vconn.GetSecondWriteFlow()
					gress.Rate = w

					ma[f.(string)] = gress
				} else {
					utils.DebugLog("no link》》》》》》》》》》》》》》》》》》》》》")
					gress.Rate = 0
					ma[f.(string)] = gress
				}
			}

		}
		m := make(map[string]interface{}, 0)
		m["taskList"] = ma
		w.Write(httpserv.NewJson(m, setting.SUCCESSCode, "request success").ToBytes())
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "fileHash is required").ToBytes())
	}
}
