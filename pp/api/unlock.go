package api

import (
	"github.com/qsnetwork/sds/pp/peers"
	"github.com/qsnetwork/sds/pp/setting"
	"github.com/qsnetwork/sds/utils/httpserv"
	"net/http"
	"time"
)

func login(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, false)
	if err != nil {
		return
	}
	walletAddress := ""
	password := ""
	if data["walletAddress"] != nil {
		walletAddress = data["walletAddress"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "walletAddress is required").ToBytes())
		return
	}

	if data["password"] != nil {
		password = data["password"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "password is required").ToBytes())
		return
	}

	err = peers.Login(walletAddress, password)
	if err != nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, err.Error()).ToBytes())
	} else {
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
}
