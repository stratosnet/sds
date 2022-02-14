package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"sort"

	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/httpserv"
)

// getWalletList GET
func getWalletList(w http.ResponseWriter, request *http.Request) {
	files, err := ioutil.ReadDir(setting.Config.AccountDir)
	if err != nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "no local wallet").ToBytes())
		return
	}
	walletlist := make([]walletList, 0)
	fileArr := make([]string, 0)
	balanceArr := make([]interface{}, 0)
	for _, info := range files {
		fileArr = append(fileArr, info.Name())
	}
	//req := make(map[string][]string, 0)
	//req["addresses"] = fileArr
	//reqByte, _ := json.Marshal(req)
	//js, err := httprequest("POST", setting.Config.BPURL+"/account/balance/batch", strings.NewReader(string(reqByte)))
	//if err != nil {
	//	utils.ErrorLog("err", err)
	//	// w.Write(httpserv.NewJson(nil, setting.FAILCode, "failed to get balance").ToBytes())
	//	// return
	//}
	//utils.DebugLog("BPURL", js)
	//if js.Data["list"] != nil {
	//	balanceArr = js.Data["list"].([]interface{})
	//}
	data := make(map[string]interface{})
	for _, info := range files {
		// utils.DebugLog("BPURL", setting.Config.BPURL, info.ModTime().Unix())
		if info.Name() == ".DS_Store" {
			continue
		}
		var state bool
		if setting.WalletAddress == info.Name() {
			state = true
		}
		f, err := os.Open(setting.Config.AccountDir + "/" + info.Name())
		if err != nil {
			utils.ErrorLog("open err", err)
		}
		contents, _ := ioutil.ReadAll(f)
		json.Unmarshal(contents, &data)
		name := ""
		if data["account"] != nil {
			name = data["account"].(string)
		}
		for _, li := range balanceArr {
			m := li.(map[string]interface{})
			if m["address"].(string) == info.Name() {
				wallet := walletList{
					WalletName:       name,
					WalletAddress:    info.Name(),
					State:            state,
					ModificationTime: info.ModTime().Unix(),
					Balance:          m["balance"].(float64),
				}
				walletlist = append(walletlist, wallet)
				break
			}
		}
		if len(balanceArr) == 0 {
			wallet := walletList{
				WalletName:       name,
				WalletAddress:    info.Name(),
				State:            state,
				ModificationTime: info.ModTime().Unix(),
				Balance:          0,
			}
			walletlist = append(walletlist, wallet)
		}
	}
	sort.Sort(MsgSlice(walletlist))
	list := make(map[string][]walletList, 0)
	list["list"] = walletlist
	w.Write(httpserv.NewJson(list, setting.SUCCESSCode, "request success").ToBytes())
}

// MsgSlice sort based on MsgSlice.count decreasingly
type MsgSlice []walletList

func (a MsgSlice) Len() int {
	return len(a)
}
func (a MsgSlice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a MsgSlice) Less(i, j int) bool {
	return a[j].ModificationTime < a[i].ModificationTime
}
