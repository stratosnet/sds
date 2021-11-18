package api

import (
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/httpserv"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
)

// changePassword  POST, params: (oldPassWord，newPassWord，againPassWord，walletAddress)
func changePassword(w http.ResponseWriter, request *http.Request) {
	dir := setting.Config.AccountDir
	data, err := HTTPRequest(request, w, false)
	if err != nil {
		return
	}
	oldPassWord := ""
	newPassWord := ""
	againPassWord := ""
	walletAddress := ""
	exp1 := regexp.MustCompile(`^[A-Za-z0-9]{8,16}$`)
	if data["oldPassword"] != nil {
		oldPassWord = data["oldPassword"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "old password is required").ToBytes())
		return
	}

	if data["newPassword"] != nil {
		newPassWord = data["newPassword"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "new password is required").ToBytes())
		return
	}

	if exp1.FindAllString(newPassWord, -1) == nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "8-16characters,include letter and number").ToBytes())
		return
	}

	if data["againPassword"] != nil {
		againPassWord = data["againPassword"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "new password is required again for confirmation").ToBytes())
		return
	}

	if data["walletAddress"] != nil {
		walletAddress = data["walletAddress"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "wallet address is required").ToBytes())
		return
	}
	keyjson, err := ioutil.ReadFile(filepath.Join(dir, walletAddress+".json"))
	if err != nil {
		utils.ErrorLog("readfile err", err)
		httpserv.NewJson(nil, setting.FAILCode, "wallet not exist").ToBytes()
		return
	}
	key, err := utils.DecryptKey(keyjson, oldPassWord)
	if err != nil {
		utils.ErrorLog("getPublicKey DecryptKey", err)
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "wrong old password").ToBytes())
		return
	}
	if newPassWord != againPassWord {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "new password not match").ToBytes())
		return
	}
	err = utils.ChangePassword(walletAddress, dir, newPassWord, key)
	if err != nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "failed to change password").ToBytes())
		return
	}
	w.Write(httpserv.NewJson(nil, setting.SUCCESSCode, "password changed successfully").ToBytes())
}
