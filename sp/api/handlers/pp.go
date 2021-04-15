package handlers

import (
	"encoding/json"
	"github.com/stratosnet/sds/sp/api/core"
	"github.com/stratosnet/sds/sp/common"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils/database"
	"net/http"

	"github.com/gorilla/mux"
)

// PP 用户接口
type PP struct {
	server *core.APIServer
}

// GetAPIServer 获取API服务实例
func (e *PP) GetAPIServer() *core.APIServer {
	return e.server
}

// SetAPIServer 设置API服务实例
func (e *PP) SetAPIServer(server *core.APIServer) {
	e.server = server
}

// List pp列表
func (e *PP) List(params map[string]interface{}, r *http.Request) ([]map[string]interface{}, int, string) {

	data := make([]map[string]interface{}, 0)

	res, err := e.GetAPIServer().DB.FetchTables([]table.PP{}, map[string]interface{}{})
	if err == nil {
		ppList := res.([]table.PP)
		if len(ppList) > 0 {
			for _, pp := range ppList {
				data = append(data, database.Table2Map(&pp))
			}
		}
	}

	return data, 200, "ok"
}

// Backup 备份pp
func (e *PP) Backup(params map[string]interface{}, r *http.Request) ([]map[string]interface{}, int, string) {

	vals := mux.Vars(r)

	if walletAddress, ok := vals["wa"]; ok {

		data := make([]map[string]interface{}, 0)

		msg := &common.MsgBackupPP{WalletAddress: walletAddress}

		msgJson, _ := json.Marshal(msg)
		e.GetAPIServer().Cache.EnQueue("msg_queue", msgJson)

		return data, 200, "ok"
	}

	return nil, 400, "参数错误"
}
