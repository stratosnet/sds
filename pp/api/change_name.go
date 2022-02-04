package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/httpserv"
)

func changeName(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, false)
	if err != nil {
		return
	}
	walletAddress := ""
	name := ""
	if data["walletAddress"] != nil {
		walletAddress = data["walletAddress"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "walletAddress is required").ToBytes())
		return
	}
	if data["name"] != nil {
		name = data["name"].(string)
	}
	js := make(map[string]interface{}, 0)
	path := filepath.Join(setting.Config.AccountDir, walletAddress)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND, 0777)
	if err != nil {
		utils.ErrorLog("open err", err)
	}
	contents, _ := ioutil.ReadAll(f)
	json.Unmarshal(contents, &js)
	oldName := ""
	fStr := string(contents)
	if os.Truncate(f.Name(), 0) != nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "error change wallet name").ToBytes())
		return
	}
	if js["account"] != nil {
		oldName = js["account"].(string)
		fStr = strings.Replace(fStr, oldName, name, -1)
		_, err2 := f.WriteString(fStr)
		if err2 != nil {
			utils.ErrorLog("err2>>>>>", err2)
			w.Write(httpserv.NewJson(nil, setting.FAILCode, "error change wallet name").ToBytes())
			return
		}
	} else {
		js["account"] = name
		new, err := json.Marshal(&js)
		if err != nil {
			w.Write(httpserv.NewJson(nil, setting.FAILCode, "error change wallet name").ToBytes())
			return
		}
		utils.DebugLog(">>>>>>>>", string(new))
		_, err = f.WriteString(string(new))
		if err != nil {
			w.Write(httpserv.NewJson(nil, setting.FAILCode, "error change wallet name").ToBytes())
			return
		}
	}
	w.Write(httpserv.NewJson(nil, setting.SUCCESSCode, "error change wallet name").ToBytes())
}
