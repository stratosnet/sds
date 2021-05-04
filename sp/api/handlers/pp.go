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

// PP user interface
type PP struct {
	server *core.APIServer
}

// GetAPIServer get API instance
func (e *PP) GetAPIServer() *core.APIServer {
	return e.server
}

// SetAPIServer set API instance
func (e *PP) SetAPIServer(server *core.APIServer) {
	e.server = server
}

// List pp list
func (e *PP) List(params map[string]interface{}, r *http.Request) ([]map[string]interface{}, int, string) {

	data := make([]map[string]interface{}, 0)

	res, err := e.GetAPIServer().DB.FetchTables([]table.PP{}, map[string]interface{}{})
	if err != nil {
		return data, 200, "ok"

	}
	ppList := res.([]table.PP)
	for _, pp := range ppList {
		data = append(data, database.Table2Map(&pp))
	}

	return data, 200, "ok"
}

// Backup backup pp
func (e *PP) Backup(params map[string]interface{}, r *http.Request) ([]map[string]interface{}, int, string) {

	vals := mux.Vars(r)

	walletAddress, ok := vals["wa"]
	if !ok {
		return nil, 400, "invalid parameter"
	}

	data := make([]map[string]interface{}, 0)

	msg := &common.MsgBackupPP{WalletAddress: walletAddress}

	msgJson, _ := json.Marshal(msg)
	e.GetAPIServer().Cache.EnQueue("msg_queue", msgJson)

	return data, 200, "ok"
}
