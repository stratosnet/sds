package api

import (
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"

	"github.com/stratosnet/sds/pp/serv"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/httpserv"
	"github.com/stratosnet/stratos-chain/types"
)

// createWallet POST, params：(password，againpassword，name, mnemonic, passphrase, hdpath)
func createWallet(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, false)
	if err != nil {
		return
	}
	files, _ := ioutil.ReadDir(setting.Config.AccountDir)
	password := ""
	againPassword := ""
	exp1 := regexp.MustCompile(`^[A-Za-z0-9]{8,16}$`)
	name := "Wallet" + strconv.Itoa(len(files))
	mnemonic := ""
	hdPath := ""
	if data["password"] != nil {
		password = data["password"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "password is required").ToBytes())
		return
	}
	if data["againPassword"] != nil {
		againPassword = data["againPassword"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "password is required again for confirmation").ToBytes())
		return
	}
	if data["name"] != nil {
		name = data["name"].(string)
	}
	if data["mnemonic"] != nil {
		mnemonic = data["mnemonic"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "mnemonic is required").ToBytes())
		return
	}
	//if data["passphrase"] != nil {
	//	passphrase = data["passphrase"].(string)
	//} else {
	//	w.Write(httpserv.NewJson(nil, setting.FAILCode, "bip39 passphrase is required").ToBytes())
	//	return
	//}
	if data["hdpath"] != nil {
		hdPath = data["hdpath"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "hdpath is required").ToBytes())
		return
	}
	if exp1.FindAllString(password, -1) == nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "8-16characters,include letter and number").ToBytes())
		return
	}
	if password != againPassword {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "password doesn't match").ToBytes())
		return
	}
	walletAddress, err := utils.CreateWallet(setting.Config.AccountDir, name, password, types.StratosBech32Prefix,
		mnemonic, "", hdPath)
	if err != nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "failed to create wallet").ToBytes())
		return
	}
	walletAddressString, err := walletAddress.ToBech(types.StratosBech32Prefix)
	if err != nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "failed to convert wallet address to bech32 string").ToBytes())
		return
	}

	data1 := walletInfo{
		WalletInfo: walletList{
			WalletAddress: walletAddressString,
			Balance:       0,
		},
	}
	utils.DebugLog("add", data1)
	setting.WalletAddress = walletAddressString
	serv.Login(setting.WalletAddress, password)
	w.Write(httpserv.NewJson(data1, setting.SUCCESSCode, "create wallet successfully").ToBytes())
}
