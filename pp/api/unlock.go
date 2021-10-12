package api

import (
	"net/http"
	"time"

	"github.com/stratosnet/sds/pp/serv"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"
)

func login(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, false)
	if err != nil {
		return
	}
	walletAddress := ""
	password := ""
	if data["walletAddress"] == nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "walletAddress is required").ToBytes())
		return
	}
	walletAddress = data["walletAddress"].(string)

	if data["password"] == nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "password is required").ToBytes())
		return
	}
	password = data["password"].(string)

	err = serv.Login(walletAddress, password)
	if err != nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, err.Error()).ToBytes())
		return
	}
	start := time.Now().Unix()
	for {
		if setting.IsLoad {
			w.Write(httpserv.NewJson(nil, setting.SUCCESSCode, "login successfully").ToBytes())
			return
		}
		if time.Now().Unix()-start > 10 {
			w.Write(httpserv.NewJson(nil, setting.FAILCode, "login failed, timeout").ToBytes())
			return
		}
	}
}
