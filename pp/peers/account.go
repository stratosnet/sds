package peers

import (
	"errors"
	"fmt"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto"
	"io/ioutil"
)

// CreateAccount
func CreateAccount(password, name string) string {
	account, err := utils.CreateAccount(setting.Config.AccountDir, name, password, setting.Config.ScryptN, setting.Config.ScryptP)
	if utils.CheckError(err) {
		utils.ErrorLog("CreateAccount error", err)
		return ""
	}
	setting.WalletAddress = account.String()
	getPublickKey(setting.Config.AccountDir+"/"+setting.WalletAddress, password)
	utils.Log("Create account success ,", setting.WalletAddress)
	if setting.NetworkAddress != "" {
		InitPeer()
	}
	return setting.WalletAddress
}

// GetWalletAddress
func GetWalletAddress() {
	files, err := ioutil.ReadDir(setting.Config.AccountDir)
	if len(files) == 0 {
		// CreateAccount(setting.Config.DefPassword)
		return
	}
	if utils.CheckError(err) {
		// CreateAccount(setting.Config.DefPassword)
		return
	}
	setting.WalletAddress = files[0].Name()
	getPublickKey(setting.Config.AccountDir+"/"+setting.WalletAddress, setting.Config.DefPassword)
	utils.Log("setting.WalletAddress,", setting.WalletAddress)

}

func getPublickKey(filePath, password string) bool {
	keyjson, err := ioutil.ReadFile(filePath)
	if utils.CheckError(err) {
		fmt.Println("getPublickKey ioutil.ReadFile", err)
		return false
	}
	key, err := utils.DecryptKey(keyjson, password)

	if utils.CheckError(err) {
		fmt.Println("getPublickKey DecryptKey", err)
		return false
	}
	setting.PrivateKey = key.PrivateKey
	setting.PublickKey = crypto.FromECDSAPub(&key.PrivateKey.PublicKey)
	utils.DebugLog("publicKey", setting.PublickKey)
	fmt.Println("unlock wallet successfully ", setting.WalletAddress)
	return true
}

// Accounts get all accounts
func Accounts() {
	files, _ := ioutil.ReadDir(setting.Config.AccountDir)
	if len(files) == 0 {
		fmt.Println("no account exist yet")

	} else {
		for _, info := range files {
			fmt.Println(info.Name())
		}
	}
}

// NewAccount
func NewAccount(password, name string) {
	if password == "" {
		fmt.Println("input password")
	} else {
		CreateAccount(password, name)
	}
}

// Login
func Login(account, password string) error {
	utils.DebugLog("account = ", account)
	// utils.DebugLog("password = ", password)
	if account == "" {
		fmt.Println("please input wallet address")
		return errors.New("please input wallet address")
	}
	if password == "" {
		fmt.Println("please input password")
		return errors.New("please input password")
	}

	files, _ := ioutil.ReadDir(setting.Config.AccountDir)
	if len(files) == 0 {
		fmt.Println("wrong account or password")
		return errors.New("wrong account or password")
	}
	for _, info := range files {
		if info.Name() == ".placeholder" || info.Name() != account {
			continue
		}
		utils.Log(info.Name())
		if getPublickKey(setting.Config.AccountDir+"/"+account, password) {
			setting.WalletAddress = account
			InitPeer()
			return nil
		}
		fmt.Println("wrong password")
		return errors.New("wrong password")
	}
	fmt.Println("wrong account or password")
	return errors.New("wrong account or password")
}
