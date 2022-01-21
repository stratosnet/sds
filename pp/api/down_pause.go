package api

import (
	"net/http"

	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"

	"github.com/google/uuid"
)

func downPause(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	if data["tasks"] != nil {
		fileHash := data["tasks"].([]interface{})
		type pause struct {
			TaskID string `json:"taskID"`
			State  bool   `json:"state"`
		}
		list := make([]*pause, 0)
		for _, f := range fileHash {
			l := &pause{
				TaskID: f.(string),
				State:  true,
			}
			list = append(list, l)
			if val, ok := setting.DownloadTaskIDMap.Load(f.(string)); ok {
				event.DownloadSlicePause(val.(string), uuid.New().String(), w)
			}
		}
		result := make(map[string][]*pause, 0)
		result["list"] = list
		w.Write(httpserv.NewJson(result, setting.SUCCESSCode, "pause successfully").ToBytes())
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "fileHash is required").ToBytes())
	}

}
