package handlers

import (
	"net/http"
	"github.com/qsnetwork/sds/sp/api/core"
)

type Data struct {
	server *core.APIServer
}

//func (ds *Data) FreshPage() {

//c := clock.NewClock()
//c.AddJobRepeat(ds.FreshTime, 0, func() {
//
//	data := make(map[string]interface{})
//
//	rows, err := ds.driver.FetchAll("v_upload_file", map[string]interface{}{})
//	if err == nil {
//		data["upload_file"] = rows
//	}
//
//	rows, err = ds.driver.FetchAll("v_upload_statistics", map[string]interface{}{})
//	if err == nil {
//		data["upload_statistics"] = rows
//	}
//
//	rows, err = ds.driver.FetchAll("v_download_statistics", map[string]interface{}{})
//	if err == nil {
//		data["download_statistics"] = rows
//	}
//
//	j, err := json.Marshal(data)
//	if err != nil {
//		utils.ErrorLog(err)
//	}
//
//	ds.CacheData.Store("cache_data", j)
//})
//}

// GetAPIServer 获取API服务实例
func (e *Data) GetAPIServer() *core.APIServer {
	return e.server
}

// SetAPIServer 设置API服务实例
func (e *Data) SetAPIServer(server *core.APIServer) {
	e.server = server
}

// Statistics 获取统计数据
func (e *Data) Statistics(params map[string]interface{}, r *http.Request) (map[string]interface{}, int, string) {

	data := map[string]interface{}{}

	return data, 100, "ok"
}
