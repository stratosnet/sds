package api

import (
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/httpserv"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
)

// createwallet POST, params：(password，againpassword，(walletname))
func createWallet(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, false)
	if err != nil {
		return
	}
	files, _ := ioutil.ReadDir(setting.Config.AccountDir)
	password := ""
	againPassWord := ""
	exp1 := regexp.MustCompile(`^[A-Za-z0-9]{8,16}$`)
	name := "Account" + strconv.Itoa(len(files))
	if data["password"] != nil {
		password = data["password"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "password is required").ToBytes())
		return
	}
	if data["againPassword"] != nil {
		againPassWord = data["againPassword"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "password is required again for confirmation").ToBytes())
		return
	}
	if data["name"] != nil {
		name = data["name"].(string)
	}
	// walletname := request.PostFormValue("walletname")
	if exp1.FindAllString(password, -1) == nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "8-16characters,include letter and number").ToBytes())
		return
	}
	if password != againPassWord {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "password doesn't match").ToBytes())
		return
	}
	account, err := utils.CreateAccount(setting.Config.AccountDir, name, password, setting.Config.ScryptN, setting.Config.ScryptP)
	if err != nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "failed to create wallet").ToBytes())
		return
	}
	accountString, err := account.ToBech(setting.Config.AddressPrefix)
	if err != nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "failed to convert wallet address to string: "+err.Error()).ToBytes())
		return
	}

	data1 := walletInfo{
		WalletInfo: walletList{
			WalletAddress: accountString,
			Balance:       0,
		},
	}
	utils.DebugLog("add", data1)
	setting.WalletAddress = accountString
	peers.Login(setting.WalletAddress, password)
	w.Write(httpserv.NewJson(data1, setting.SUCCESSCode, "create wallet successfully").ToBytes())
}
