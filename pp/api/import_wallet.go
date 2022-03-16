package api

import (
	"net/http"
	"path/filepath"

	"github.com/stratosnet/sds/pp/serv"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	"github.com/stratosnet/sds/utils/httpserv"
	"github.com/stratosnet/stratos-chain/types"
)

// importWallet POST, params：(keystore , password)
func importWallet(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, false)
	if err != nil {
		return
	}
	keystore := ""
	password := ""
	if data["keystore"] != nil {
		keystore = data["keystore"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "keystore is required").ToBytes())
		return
	}

	if data["password"] != nil {
		password = data["password"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "password is required").ToBytes())
		return
	}

	dir := setting.Config.AccountDir
	key, err := utils.DecryptKey([]byte(keystore), password)

	if utils.CheckError(err) {
		utils.ErrorLog("getPublicKey DecryptKey", err)
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "wrong password").ToBytes())
		return
	}
	setting.WalletPrivateKey = key.PrivateKey
	setting.WalletPublicKey = secp256k1.PrivKeyToPubKey(key.PrivateKey)
	setting.WalletAddress, err = key.Address.ToBech(types.StratosBech32Prefix)
	if err != nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "failed to convert wallet address to bech32 string").ToBytes())
		return
	}
	ks := utils.GetKeyStorePassphrase(dir)
	filename := filepath.Join(dir, setting.WalletAddress+".json")
	err = ks.StoreKey(filename, key, password)
	if err != nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "failed to import wallet").ToBytes())
		return
	}
	//utils.DebugLog("BPURL", setting.Config.BPURL)
	//js, err := httprequest("GET", setting.Config.BPURL+"/account/balance?address="+setting.WalletAddress, nil)
	//if err != nil {
	//
	//	utils.ErrorLog("failed to get balance")
	//}
	//utils.DebugLog("BPURL end", setting.Config.BPURL)
	//var balance float64
	//if js.Data["balance"] != nil {
	//	balance = js.Data["balance"].(float64)
	//}
	data1 := walletInfo{
		WalletInfo: walletList{
			WalletName:    key.Name,
			WalletAddress: setting.WalletAddress,
			//Balance:       balance,
			State: true,
		},
	}
	w.Write(httpserv.NewJson(data1, setting.SUCCESSCode, "successfully import wallet").ToBytes())
	serv.Login(setting.WalletAddress, password)
}
