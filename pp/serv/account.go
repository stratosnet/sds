package serv

import (
	"errors"
	"io/fs"
	"io/ioutil"
	"path/filepath"

	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	"github.com/stratosnet/stratos-chain/types"
)

// CreateWallet
func CreateWallet(password, name, mnemonic, hdPath string) string {
	if mnemonic == "" {
		newMnemonic, err := utils.NewMnemonic()
		if err != nil {
			utils.ErrorLog("Couldn't generate new mnemonic", err)
			return ""
		}
		mnemonic = newMnemonic
	}
	account, err := utils.CreateWallet(setting.Config.AccountDir, name, password, types.StratosBech32Prefix,
		mnemonic, "", hdPath)
	if utils.CheckError(err) {
		utils.ErrorLog("CreateWallet error", err)
		return ""
	}
	setting.WalletAddress, err = account.ToBech(types.StratosBech32Prefix)
	if utils.CheckError(err) {
		utils.ErrorLog("CreateWallet error", err)
		return ""
	}
	if setting.WalletAddress != "" {
		setting.SetConfig("WalletAddress", setting.WalletAddress)
	}
	getPublicKey(filepath.Join(setting.Config.AccountDir, setting.WalletAddress+".json"), password)
	utils.Log("Create account success ,", setting.WalletAddress)
	return setting.WalletAddress
}

// GetWalletAddress
func GetWalletAddress() error {
	files, err := ioutil.ReadDir(setting.Config.AccountDir)

	if len(files) == 0 {
		return errors.New("account Dir is empty")
	}
	if err != nil {
		return err
	}

	walletAddress := setting.Config.WalletAddress
	password := setting.Config.WalletPassword
	fileName := walletAddress + ".json"

	for _, info := range files {
		if info.Name() == ".placeholder" || info.Name() != fileName {
			continue
		}
		utils.Log(info.Name())
		if getPublicKey(filepath.Join(setting.Config.AccountDir, fileName), password) {
			setting.WalletAddress = walletAddress
			return nil
		}
		return errors.New("wrong password")
	}
	return errors.New("could not find the account file corresponds to the configured wallet address")
}

func getPublicKey(filePath, password string) bool {
	keyjson, err := ioutil.ReadFile(filePath)
	if utils.CheckError(err) {
		utils.ErrorLog("getPublicKey ioutil.ReadFile", err)
		return false
	}
	key, err := utils.DecryptKey(keyjson, password)

	if utils.CheckError(err) {
		utils.ErrorLog("getPublicKey DecryptKey", err)
		return false
	}
	setting.WalletPrivateKey = key.PrivateKey
	setting.WalletPublicKey = secp256k1.PrivKeyToPubKey(key.PrivateKey)
	utils.DebugLog("publicKey", setting.WalletPublicKey)
	utils.Log("unlock wallet successfully ", setting.WalletAddress)
	return true
}

// Wallets get all wallets
func Wallets() {
	files, _ := ioutil.ReadDir(setting.Config.AccountDir)
	var wallets []string
	for _, file := range files {
		fileName := file.Name()
		if fileName[len(fileName)-5:] == ".json" && fileName[:len(types.SdsNodeP2PAddressPrefix)] != types.SdsNodeP2PAddressPrefix {
			wallets = append(wallets, fileName[:len(fileName)-5])
		}
	}

	if len(wallets) == 0 {
		utils.Log("no wallet exists yet")
	} else {
		for _, wallet := range wallets {
			utils.Log(wallet)
		}
	}
}

// Login
func Login(walletAddress, password string) error {
	files, err := GetWallets(walletAddress, password)
	if err != nil {
		return err
	}
	fileName := walletAddress + ".json"
	for _, info := range files {
		if info.Name() == ".placeholder" || info.Name() != fileName {
			continue
		}
		utils.Log(info.Name())
		if getPublicKey(filepath.Join(setting.Config.AccountDir, fileName), password) {
			setting.SetConfig("WalletAddress", walletAddress)
			setting.SetConfig("WalletPassword", password)
			setting.WalletAddress = walletAddress
			peers.InitPeer(event.RegisterEventHandle)
			return nil
		}
		utils.ErrorLog("wrong password")
		return errors.New("wrong password")
	}
	utils.ErrorLog("wrong walletAddress or password")
	return errors.New("wrong walletAddress or password")
}

func GetWallets(walletAddress string, password string) ([]fs.FileInfo, error) {
	utils.DebugLog("walletAddress = ", walletAddress)
	if walletAddress == "" {
		utils.ErrorLog("please input wallet address")
		return nil, errors.New("please input wallet address")
	}
	if password == "" {
		utils.ErrorLog("please input password")
		return nil, errors.New("please input password")
	}

	files, _ := ioutil.ReadDir(setting.Config.AccountDir)
	if len(files) == 0 {
		utils.ErrorLog("wrong account or password")
		return nil, errors.New("wrong account or password")
	}
	return files, nil
}
