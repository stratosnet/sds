package handlers

import (
	"net/http"
	"github.com/qsnetwork/qsds/sp/api/core"
)

// User 用户接口
type User struct {
	server *core.APIServer
}

// GetAPIServer 获取API服务实例
func (e *User) GetAPIServer() *core.APIServer {
	return e.server
}

// SetAPIServer 设置API服务实例
func (e *User) SetAPIServer(server *core.APIServer) {
	e.server = server
}

// Login 登录
func (e *User) Login(params map[string]interface{}, r *http.Request) (map[string]interface{}, int, string) {

	data := map[string]interface{}{
		"token": "112233",
	}

	return data, 200, "ok"
}
