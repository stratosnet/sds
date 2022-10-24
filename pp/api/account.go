package api

import (
	"context"
	"errors"
	"io/fs"
	"io/ioutil"
	"path/filepath"

	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	"github.com/stratosnet/stratos-chain/types"
)

// CreateWallet
func CreateWallet(ctx context.Context, password, name, mnemonic, hdPath string) string {
	if mnemonic == "" {
		newMnemonic, err := utils.NewMnemonic()
		if err != nil {
			pp.ErrorLog(ctx, "Couldn't generate new mnemonic", err)
			return ""
		}
		mnemonic = newMnemonic
	}
	account, err := utils.CreateWallet(setting.Config.AccountDir, name, password, types.StratosBech32Prefix,
		mnemonic, "", hdPath)
	if utils.CheckError(err) {
		pp.ErrorLog(ctx, "CreateWallet error", err)
		return ""
	}
	setting.WalletAddress, err = account.ToBech(types.StratosBech32Prefix)
	if utils.CheckError(err) {
		pp.ErrorLog(ctx, "CreateWallet error", err)
		return ""
	}
	if setting.WalletAddress != "" {
		setting.SetConfig("wallet_address", setting.WalletAddress)
	}
	getPublicKey(ctx, filepath.Join(setting.Config.AccountDir, setting.WalletAddress+".json"), password)
	utils.Log("Create account success ,", setting.WalletAddress)
	return setting.WalletAddress
}

// GetWalletAddress
func GetWalletAddress(ctx context.Context) error {
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
		if getPublicKey(ctx, filepath.Join(setting.Config.AccountDir, fileName), password) {
			setting.WalletAddress = walletAddress
			return nil
		}
		return errors.New("wrong password")
	}
	return errors.New("could not find the account file corresponds to the configured wallet address")
}

func getPublicKey(ctx context.Context, filePath, password string) bool {
	keyjson, err := ioutil.ReadFile(filePath)
	if utils.CheckError(err) {
		pp.ErrorLog(ctx, "getPublicKey ioutil.ReadFile", err)
		return false
	}
	key, err := utils.DecryptKey(keyjson, password)

	if utils.CheckError(err) {
		pp.ErrorLog(ctx, "getPublicKey DecryptKey", err)
		return false
	}
	setting.WalletPrivateKey = key.PrivateKey
	setting.WalletPublicKey = secp256k1.PrivKeyToPubKey(key.PrivateKey).Bytes()
	pp.DebugLog(ctx, "publicKey", setting.WalletPublicKey)
	pp.Log(ctx, "unlock wallet successfully ", setting.WalletAddress)
	return true
}

// Wallets get all wallets
func Wallets(ctx context.Context) []string {
	files, _ := ioutil.ReadDir(setting.Config.AccountDir)
	var wallets []string
	for _, file := range files {
		fileName := file.Name()
		if fileName[len(fileName)-5:] == ".json" && fileName[:len(types.SdsNodeP2PAddressPrefix)] != types.SdsNodeP2PAddressPrefix {
			wallets = append(wallets, fileName[:len(fileName)-5])
		}
	}

	if len(wallets) == 0 {
		pp.Log(ctx, "no wallet exists yet")
	} else {
		for _, wallet := range wallets {
			pp.Log(ctx, wallet)
		}
	}
	return wallets
}

// Login
func Login(ctx context.Context, walletAddress, password string) error {
	files, err := GetWallets(ctx, walletAddress, password)
	if err != nil {
		return err
	}
	fileName := walletAddress + ".json"
	for _, info := range files {
		if info.Name() == ".placeholder" || info.Name() != fileName {
			continue
		}
		utils.Log(info.Name())
		if getPublicKey(ctx, filepath.Join(setting.Config.AccountDir, fileName), password) {
			setting.SetConfig("wallet_address", walletAddress)
			setting.SetConfig("wallet_password", password)
			setting.WalletAddress = walletAddress
			network.GetPeer(ctx).InitPeer(ctx)
			return nil
		}
		pp.ErrorLog(ctx, "wrong password")
		return errors.New("wrong password")
	}
	pp.ErrorLog(ctx, "wrong walletAddress or password")
	return errors.New("wrong walletAddress or password")
}

func GetWallets(ctx context.Context, walletAddress string, password string) ([]fs.FileInfo, error) {
	pp.DebugLog(ctx, "walletAddress = ", walletAddress)
	if walletAddress == "" {
		pp.ErrorLog(ctx, "please input wallet address")
		return nil, errors.New("please input wallet address")
	}

	files, _ := ioutil.ReadDir(setting.Config.AccountDir)
	if len(files) == 0 {
		pp.ErrorLog(ctx, "wrong account or password")
		return nil, errors.New("wrong account or password")
	}
	return files, nil
}
