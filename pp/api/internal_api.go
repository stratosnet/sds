package api

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/httpserv"
)

var (
	bpurl string
)

// walletList local wallet list return format
type walletList struct {
	WalletName       string  `json:"walletName"`
	WalletAddress    string  `json:"walletAddress"`
	Balance          float64 `json:"balance"`
	State            bool    `json:"state"`
	ModificationTime int64   `json:"modificationTime"`
}

type walletInfo struct {
	WalletInfo walletList `json:"walletInfo"`
}

// httpres http response format
type httpres struct {
	Errcode int                    `json:"errcode"`
	Data    map[string]interface{} `json:"data"`
	Message string                 `json:"message"`
}

// StartHTTPServ
func StartHTTPServ() {
	httpServ := httpserv.MyNewHTTPServ(setting.Config.InternalPort)
	httpServ.MyRoute("/unlock", login)
	httpServ.MyRoute("/walletList", getWalletList)
	httpServ.MyRoute("/createWallet", createWallet)
	httpServ.MyRoute("/importWallet", importWallet)
	httpServ.MyRoute("/changePassword", changePassword)
	httpServ.MyRoute("/getAllFile", getAllFile)
	httpServ.MyRoute("/getShareLink", getAllShareLink)
	httpServ.MyRoute("/downloadFile", downloadFile)
	httpServ.MyRoute("/uploadFile", uploadFile)
	httpServ.MyRoute("/deleteFile", deleteFile)
	httpServ.MyRoute("/setConfig", setConfig)
	httpServ.MyRoute("/getKeyStore", getKeyStore)
	httpServ.MyRoute("/shareFile", shareFile)
	httpServ.MyRoute("/downPause", downPause)
	httpServ.MyRoute("/deleteShare", deleteShare)
	httpServ.MyRoute("/getShareFile", getShareFile)
	httpServ.MyRoute("/downloadCancel", downloadCancel)
	httpServ.MyRoute("/upPause", upPause)
	httpServ.MyRoute("/upCancel", upCancel)
	httpServ.MyRoute("/upProgress", upProgress)
	httpServ.MyRoute("/downProgress", downProgress)
	httpServ.MyRoute("/closeHTTP", closeHTTP)
	httpServ.MyRoute("/legal", legal)
	httpServ.MyRoute("/changeName", changeName)
	httpServ.MyRoute("/streamVideoStorageInfo/", streamVideoInfoCache)
	httpServ.MyRoute("/streamVideo/", streamVideoP2P)
	httpServ.MyRoute("/streamVideoStorageInfoHttp/", streamVideoInfoHttp)
	httpServ.MyRoute("/streamVideoHttp/", streamVideoHttp)
	httpServ.MyRoute("/clearStreamTask/", clearStreamTask)
	httpServ.MyStart()
}

// httprequest go
func httprequest(method, url string, postData io.Reader) (js httpres, err error) {

	req, err := http.NewRequest(method, url, postData)
	if err != nil {
		utils.ErrorLog("err ", err)
		return
	}
	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		utils.ErrorLog("err ", err)
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		utils.ErrorLog("err", err)
	}
	js = httpres{}
	err = json.Unmarshal(body, &js)
	if err != nil {
		utils.ErrorLog("err", err)
	}
	return
}

// ParseBody
func ParseBody(r *http.Request) (map[string]interface{}, error) {

	data := make(map[string]interface{})

	log := strings.Join([]string{r.Method, r.RequestURI}, " ")

	contentType := r.Header.Get("Content-Type")
	typeString := strings.Split(contentType, ";")
	utils.DebugLog("type>>>>>", typeString)
	switch typeString[0] {

	case "application/json":
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		utils.DebugLog("body", string(body))
		if string(body) != "{}" {
			log = log + " " + string(body)
			err = json.Unmarshal(body, &data)
			if err != nil {
				return nil, errors.New("error parsing params")
			}
			r.Body.Close()
		} else {
			return nil, errors.New("empty params not allowed")
		}
	default:
		return nil, errors.New("Content-Type error")
	}

	return data, nil
}

// HTTPRequest
func HTTPRequest(request *http.Request, w http.ResponseWriter, isNeedWalletAddress bool) (data map[string]interface{}, err error) {
	data, err = ParseBody(request)
	if err != nil {
		_, _ = w.Write(httpserv.NewJson(nil, setting.FAILCode, err.Error()).ToBytes())
		return
	}
	if !isNeedWalletAddress {
		return
	}
	if setting.WalletAddress == data["walletAddress"] {
		return
	}
	err = errors.New("wallet is locked")
	_, _ = w.Write(httpserv.NewJson(nil, setting.FAILCode, "wallet is locked").ToBytes())
	return
}
