package api

import (
	"github.com/qsnetwork/qsds/pp/event"
	"github.com/qsnetwork/qsds/pp/setting"
	"github.com/qsnetwork/qsds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func downloadCancel(w http.ResponseWriter, request *http.Request) {
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
			if val, ok := setting.DownLoadTaskIDMap.Load(l.TaskID); ok {
				go event.DownloadSloceCancel(val.(string), uuid.New().String(), w)
			}
			setting.DownLoadTaskIDMap.Delete(l.TaskID)
			delete(setting.DownMap, l.TaskID)
		}
		result := make(map[string][]*pause, 0)
		result["list"] = list
		w.Write(httpserv.NewJson(result, setting.SUCCESSCode, "cancel successfully").ToBytes())
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "fileHash is required").ToBytes())
	}
}
