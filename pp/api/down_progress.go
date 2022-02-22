package api

import (
	"net/http"

	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/httpserv"
)

func downProgress(w http.ResponseWriter, request *http.Request) {
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

		type tk struct {
			taskID   string
			fileName string
			savePath string
		}
		ma := make(map[string]interface{}, 0)
		for _, f := range data["tasks"].([]interface{}) {
			m := f.(map[string]interface{})
			k := &tk{
				taskID:   m["taskID"].(string),
				fileName: m["fileName"].(string),
			}
			if m["savePath"] != nil {
				k.savePath = m["savePath"].(string)
			}
			// _, num := task.CheckDownloadOver(k.fileHash)
			// utils.Logf("downloaded：%.2f %% filehash:%s \n", (num * 100), k.fileHash)
			// if num > 1 {
			// 	utils.Log(">>>>>>>>>>>>>>>>>>>>>iuhihioioihk")
			// 	num = 1
			// }
			p := &prog{
				TaskID:   k.taskID,
				Progress: 0,
				Rate:     0,
				State:    true,
			}
			if ts, ok := setting.DownloadTaskIDMap.Load(p.TaskID); ok {
				if val, ok := setting.DownloadProgressMap.Load(ts.(string)); ok {
					p.Progress = val.(float32)
					if val.(float32) > 100 {
						p.Progress = 100
					}
				}
				if file.CheckFilePathEx(ts.(string), k.fileName, k.savePath) {
					utils.DebugLog("file downloaded")
					p.Progress = 100
					p.Rate = 0
					p.State = true
				}
				// TODO replace client.PDownloadPassageway with client.DownloadConnMap and aggregate speed of all connections that are involved in the task
				//if c, ok := client.PDownloadPassageway.Load(ts.(string)); ok {
				//	conn := c.(*cf.ClientConn)
				//	re := conn.GetSecondReadFlow()
				//	p.Rate = re
				//}
			}
			ma[k.taskID] = p
		}
		m := make(map[string]interface{}, 0)
		m["taskList"] = ma
		w.Write(httpserv.NewJson(m, setting.SUCCESSCode, "request success").ToBytes())
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "tasks is required").ToBytes())
	}
}
