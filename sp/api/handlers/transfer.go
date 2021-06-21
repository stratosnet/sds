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

type TransferRequest struct {
	SliceHash         string `json:"SliceHash"`
	FromWalletAddress string `json:"FromWalletAddress"`
	ToWalletAddress   string `json:"ToWalletAddress"`
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
	var FromWalletAddress string
	var ToWalletAddress string

	if val, ok := params["SliceHash"]; ok {
		SliceHash = val.(string)
	} else {
		return data, 400, "Invalid SliceHash"
	}

	if val, ok := params["FromWalletAddress"]; ok {
		FromWalletAddress = val.(string)
	} else {
		return data, 400, "Invalid FromWalletAddress"
	}

	if val, ok := params["ToWalletAddress"]; ok {
		ToWalletAddress = val.(string)
	} else {
		return data, 400, "Invalid ToWalletAddress"
	}

	msg := &common.MsgWrapper{
		MsgType: common.MSG_TRANSFER_NOTICE,
		Msg: &common.MsgTransferNotice{
			SliceHash:         SliceHash,
			FromWalletAddress: FromWalletAddress,
			ToWalletAddress:   ToWalletAddress,
		},
	}

	msgJson, _ := json.Marshal(msg)
	e.GetAPIServer().Cache.EnQueue("msg_queue", msgJson)

	return data, 200, "ok"
}
