package handlers

import (
	"net/http"
	"github.com/stratosnet/sds/sp/api/core"
	"github.com/stratosnet/sds/sp/storages/data"
	"github.com/stratosnet/sds/sp/storages/table"
	"strings"
	"time"
)

// Sys 系统
type Sys struct {
	server *core.APIServer
}

// GetAPIServer 获取API服务实例
func (e *Sys) GetAPIServer() *core.APIServer {
	return e.server
}

// SetAPIServer 设置API服务实例
func (e *Sys) SetAPIServer(server *core.APIServer) {
	e.server = server
}

// Setting 获取配置
func (e *Sys) Setting(params map[string]interface{}, r *http.Request) (interface{}, int, string) {

	system := &data.System{}
	if e.GetAPIServer().Load(system) == nil {
		return system, 200, "ok"
	}

	return nil, 200, "ok"
}

// Save 保存配置
func (e *Sys) Save(params map[string]interface{}, r *http.Request) (interface{}, int, string) {

	system := &data.System{}
	if e.GetAPIServer().Load(system) == nil {

		if missingBakWalletAddr, ok := params["missing_wallet_address"]; ok {

			switch missingBakWalletAddr.(type) {
			case string:
				addressInStr := missingBakWalletAddr.(string)
				if addressInStr != "" {
					addresses := strings.Split(addressInStr, ",")
					if len(addresses) > 0 {
						system.MissingBackupWalletAddr = addresses
					}
				}
			}
		}

		e.GetAPIServer().Store(system, 0)
	}

	return nil, 200, "ok"
}

// ClientDownload 下载统计
func (e *Sys) ClientDownload(params map[string]interface{}, r *http.Request) (interface{}, int, string) {

	clientDownload := &table.ClientDownloadRecord{
		Type: 1,
		Time: time.Now().Unix(),
	}
	e.GetAPIServer().DB.InsertTable(clientDownload)

	return nil, 200, "ok"
}
