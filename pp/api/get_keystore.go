package api

import (
	"io/ioutil"
	"net/http"
	"os"

	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/httpserv"
)

func getKeyStore(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, false)
	if err != nil {
		return
	}

	path := setting.Config.AccountDir
	if data["walletAddress"] != nil {
		walletAddress := data["walletAddress"].(string)
		f, err := os.Open(setting.Config.AccountDir + "/" + walletAddress)
		if err != nil {
			w.Write(httpserv.NewJson(nil, setting.FAILCode, "wallet not exist").ToBytes())
			return
		}
		contents, err := ioutil.ReadAll(f)
		if err != nil {
			w.Write(httpserv.NewJson(nil, setting.FAILCode, "failed to get keystore").ToBytes())
			return
		}
		data1 := make(map[string]string)
		data1["keystore"] = utils.ByteToString(contents)
		data1["path"] = path
		w.Write(httpserv.NewJson(data1, setting.SUCCESSCode, "successfully get keystore").ToBytes())
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "wallet address is required").ToBytes())
	}
}
