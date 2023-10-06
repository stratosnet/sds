package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/utils"
)

func sync(cmd *cobra.Command, args []string) error {
	if len(args) != 1 || len(args[0]) == 0 {
		utils.ErrorLog("wrong number of arguments")
		return nil
	}
	txHash := args[0]

	r := reqSyncMsg(txHash)
	if r == nil {
		return nil
	}
	utils.Log("- request send (method: owner_requestSync)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("json marshal error")
		return nil
	}

	// Handle rsp
	var rsp jsonrpcMessage
	err := json.Unmarshal(body, &rsp)
	if err != nil {
		return nil
	}

	var res rpc.SyncResult
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		return nil
	}
	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}
	return nil
}

func reqSyncMsg(txHash string) []byte {
	params := []rpc.ParamReqSync{{
		TxHash: txHash,
	}}

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqSync")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("owner_requestSync", pm)
}

func wrapJsonRpc(method string, param []byte) []byte {
	// compose json-rpc request
	request := &jsonrpcMessage{
		Version: "2.0",
		ID:      1,
		Method:  method,
		Params:  json.RawMessage(param),
	}
	r, e := json.Marshal(request)
	if e != nil {
		utils.ErrorLog("json marshal error", e)
		return nil
	}
	return r
}

func httpRequest(request []byte) []byte {
	if len(request) < 300 {
		utils.DebugLog("--> ", string(request))
	} else {
		utils.DebugLog("--> ", string(request[:230]), "... \"}]}")
	}

	// http post
	req, err := http.NewRequest("POST", Url, bytes.NewBuffer(request))
	if err != nil {
		panic(err)
	}
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	body, _ := io.ReadAll(resp.Body)
	if len(body) < 300 {
		utils.DebugLog("<-- ", string(body))
	} else {
		utils.DebugLog("<-- ", string(body[:230]), "... \"}]}")
	}

	resp.Body.Close()
	return body
}
