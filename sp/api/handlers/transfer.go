package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/stratosnet/sds/sp/api/core"
	"github.com/stratosnet/sds/sp/common"
)

type Transfer struct {
	server *core.APIServer
}

func (e *Transfer) GetAPIServer() *core.APIServer {
	return e.server
}

func (e *Transfer) SetAPIServer(server *core.APIServer) {
	e.server = server
}

func (e *Transfer) SliceTransfer(params map[string]interface{}, r *http.Request) ([]map[string]interface{}, int, string) {
	data := make([]map[string]interface{}, 0)
	var SliceHash string
	var FromP2PAddress string
	var ToP2PAddress string
	deleteOrigin := false

	if val, ok := params["SliceHash"]; ok {
		SliceHash = val.(string)
	} else {
		return data, 400, "Invalid SliceHash"
	}

	if val, ok := params["FromP2PAddress"]; ok {
		FromP2PAddress = val.(string)
	} else {
		return data, 400, "Invalid FromP2pAddress"
	}

	if val, ok := params["ToP2PAddress"]; ok {
		ToP2PAddress = val.(string)
	} else {
		return data, 400, "Invalid ToWalletAddress"
	}

	if val, ok := params["DeleteOrigin"]; ok {
		deleteOrigin = val.(bool)
	}

	msg := &common.MsgWrapper{
		MsgType: common.MSG_TRANSFER_NOTICE,
		Msg: &common.MsgTransferNotice{
			SliceHash:      SliceHash,
			FromP2PAddress: FromP2PAddress,
			ToP2PAddress:   ToP2PAddress,
			DeleteOrigin:   deleteOrigin,
		},
	}

	msgJson, _ := json.Marshal(msg)
	e.GetAPIServer().Cache.EnQueue("msg_queue", msgJson)

	return data, 200, "ok"
}

func (e *Transfer) SliceBackup(params map[string]interface{}, r *http.Request) ([]map[string]interface{}, int, string) {
	data := make([]map[string]interface{}, 0)
	var SliceHash string
	var FromP2PAddress string

	if val, ok := params["SliceHash"]; ok {
		SliceHash = val.(string)
	} else {
		return data, 400, "Invalid SliceHash"
	}

	if val, ok := params["FromP2PAddress"]; ok {
		FromP2PAddress = val.(string)
	} else {
		return data, 400, "Invalid FromP2PAddress"
	}

	msg := &common.MsgWrapper{
		MsgType: common.MSG_BACKUP_SLICE,
		Msg: &common.MsgBackupSlice{
			SliceHash:      SliceHash,
			FromP2PAddress: FromP2PAddress,
		},
	}

	msgJson, _ := json.Marshal(msg)
	e.GetAPIServer().Cache.EnQueue("msg_queue", msgJson)

	return data, 200, "ok"
}
