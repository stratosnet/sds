package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/stratosnet/sds/sp/api/core"
	"github.com/stratosnet/sds/sp/common"
)

type Traffic struct {
	server *core.APIServer
}

func (e *Traffic) GetAPIServer() *core.APIServer {
	return e.server
}

func (e *Traffic) SetAPIServer(server *core.APIServer) {
	e.server = server
}

func (e *Traffic) AggregateTraffic(params map[string]interface{}, r *http.Request) ([]map[string]interface{}, int, string) {
	data := make([]map[string]interface{}, 0)
	msg := &common.MsgWrapper{
		MsgType: common.MSG_AGGREGATE_TRAFFIC,
		Msg: &common.MsgAggregateTraffic{
			Time: time.Now().Unix(),
		},
	}

	msgJson, _ := json.Marshal(msg)
	e.GetAPIServer().Cache.EnQueue("msg_queue", msgJson)

	return data, 200, "ok"
}
