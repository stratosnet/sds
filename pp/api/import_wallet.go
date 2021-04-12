package api

import (
	"fmt"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto"
	"github.com/stratosnet/sds/utils/httpserv"
	"net/http"
)

// importwallet POST, params：(keystore , password)
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
		fmt.Println("getPublickKey DecryptKey", err)
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "wrong password").ToBytes())
		return
	}
	setting.PrivateKey = key.PrivateKey
	setting.PublickKey = crypto.FromECDSAPub(&key.PrivateKey.PublicKey)
	setting.WalletAddress = key.Address.String()
	ks := utils.KeyStorePassphrase{dir, setting.Config.ScryptN, setting.Config.ScryptP}
	filename := dir + "/" + key.Address.String()
	err = ks.StoreKey(filename, key, password)
	if err != nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "failed to import wallet").ToBytes())
		return
	}
	utils.DebugLog("BPURL", setting.Config.BPURL)
	js, err := httprequest("GET", setting.Config.BPURL+"/account/balance?address="+setting.WalletAddress, nil)
	if err != nil {

		utils.ErrorLog("failed to get balance")
	}
	utils.DebugLog("BPURL end", setting.Config.BPURL)
	var balance float64
	if js.Data["balance"] != nil {
		balance = js.Data["balance"].(float64)
	}
	data1 := walletInfo{
		WalletInfo: walletList{
			WalletName:    key.Account,
			WalletAddress: setting.WalletAddress,
			Balance:       balance,
			State:         true,
		},
	}
	w.Write(httpserv.NewJson(data1, setting.SUCCESSCode, "successfully import wallet").ToBytes())
	peers.Login(setting.WalletAddress, password)
}
