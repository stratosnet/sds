package handlers

import (
	"net/http"
	"github.com/qsnetwork/sds/sp/api/core"
	"github.com/qsnetwork/sds/sp/storages/table"
	"github.com/qsnetwork/sds/utils/database"

	"github.com/gorilla/mux"
)

// File 文件服务
type File struct {
	server *core.APIServer
}

// GetAPIServer 获取API服务实例
func (e *File) GetAPIServer() *core.APIServer {
	return e.server
}

// SetAPIServer 设置API服务实例
func (e *File) SetAPIServer(server *core.APIServer) {
	e.server = server
}

// List 文件列表
func (e *File) List(params map[string]interface{}, r *http.Request) ([]map[string]interface{}, int, string) {

	data := make([]map[string]interface{}, 0)

	res, err := e.GetAPIServer().DB.FetchTables([]table.File{}, map[string]interface{}{})
	if err == nil {
		files := res.([]table.File)
		if len(files) > 0 {
			for _, file := range files {
				data = append(data, database.Table2Map(&file))
			}
		}
	}

	return data, 200, "ok"
}

// Slice 切片列表
func (e *File) Slice(params map[string]interface{}, r *http.Request) ([]map[string]interface{}, int, string) {

	vals := mux.Vars(r)

	if fileHash, ok := vals["hash"]; ok {

		data := make([]map[string]interface{}, 0)

		type FileSliceGroup struct {
			table.FileSlice
			Store string
		}

		res, err := e.GetAPIServer().DB.FetchTables([]FileSliceGroup{}, map[string]interface{}{
			"columns": "*, group_concat(wallet_address, '@', network_address) as store",
			"where": map[string]interface{}{
				"file_hash = ?": fileHash,
			},
			"groupBy": "slice_hash",
			"orderBy": "slice_number ASC",
		})
		if err == nil {
			slices := res.([]FileSliceGroup)
			if len(slices) > 0 {
				for _, slice := range slices {
					data = append(data, database.Table2Map(&slice))
				}
			}
		}

		return data, 200, "ok"
	}

	return nil, 400, "参数错误"
}
