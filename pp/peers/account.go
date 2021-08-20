package peers

import (
	"errors"
	"fmt"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	"io/ioutil"
	"path/filepath"
)

// CreateWallet
func CreateWallet(password, name, mnemonic, passphrase, hdPath string) string {
	if mnemonic == "" {
		newMnemonic, err := utils.NewMnemonic()
		if err != nil {
			utils.ErrorLog("Couldn't generate new mnemonic", err)
			return ""
		}
		mnemonic = newMnemonic
	}
	account, err := utils.CreateWallet(setting.Config.AccountDir, name, password, setting.Config.AddressPrefix,
		mnemonic, passphrase, hdPath, setting.Config.ScryptN, setting.Config.ScryptP)
	if utils.CheckError(err) {
		utils.ErrorLog("CreateWallet error", err)
		return ""
	}
	setting.WalletAddress, err = account.ToBech(setting.Config.AddressPrefix)
	if utils.CheckError(err) {
		utils.ErrorLog("CreateWallet error", err)
		return ""
	}
	getPublicKey(filepath.Join(setting.Config.AccountDir, setting.WalletAddress+".json"), password)
	utils.Log("Create account success ,", setting.WalletAddress)
	if setting.NetworkAddress != "" {
		InitPeer()
	}
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
		fmt.Println("getPublicKey ioutil.ReadFile", err)
		return false
	}
	key, err := utils.DecryptKey(keyjson, password)

	if utils.CheckError(err) {
		fmt.Println("getPublicKey DecryptKey", err)
		return false
	}
	setting.WalletPrivateKey = key.PrivateKey
	setting.WalletPublicKey = secp256k1.PrivKeyToPubKey(key.PrivateKey)
	utils.DebugLog("publicKey", setting.WalletPublicKey)
	fmt.Println("unlock wallet successfully ", setting.WalletAddress)
	return true
}

// Wallets get all wallets
func Wallets() {
	files, _ := ioutil.ReadDir(setting.Config.AccountDir)
	var wallets []string
	for _, file := range files {
		fileName := file.Name()
		if fileName[len(fileName)-5:] == ".json" && fileName[:len(setting.Config.P2PKeyPrefix)] != setting.Config.P2PKeyPrefix {
			wallets = append(wallets, fileName[:len(fileName)-5])
		}
	}

	if len(wallets) == 0 {
		fmt.Println("no wallet exists yet")
	} else {
		for _, wallet := range wallets {
			fmt.Println(wallet)
		}
	}
}

// Login
func Login(walletAddress, password string) error {
	utils.DebugLog("walletAddress = ", walletAddress)
	// utils.DebugLog("password = ", password)
	if walletAddress == "" {
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
	fileName := walletAddress + ".json"
	for _, info := range files {
		if info.Name() == ".placeholder" || info.Name() != fileName {
			continue
		}
		utils.Log(info.Name())
		if getPublicKey(filepath.Join(setting.Config.AccountDir, fileName), password) {
			setting.WalletAddress = walletAddress
			InitPeer()
			return nil
		}
		fmt.Println("wrong password")
		return errors.New("wrong password")
	}
	fmt.Println("wrong walletAddress or password")
	return errors.New("wrong walletAddress or password")
}
