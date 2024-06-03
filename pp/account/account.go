package account

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/setting"
)

func CreateWallet(ctx context.Context, password, name, mnemonic, hdPath string) string {
	if mnemonic == "" {
		newMnemonic, err := fwtypes.NewMnemonic()
		if err != nil {
			pp.ErrorLog(ctx, "Couldn't generate new mnemonic", err)
			return ""
		}
		mnemonic = newMnemonic
	}
	account, created, err := fwtypes.CreateWallet(setting.Config.Home.AccountsPath, name, password, mnemonic, "", hdPath)
	if utils.CheckError(err) {
		pp.ErrorLog(ctx, "CreateWallet error", err)
		return ""
	}
	setting.WalletAddress = fwtypes.WalletAddressBytesToBech32(account.Bytes())
	if setting.WalletAddress == "" {
		pp.ErrorLog(ctx, "CreateWallet error", err)
		return ""
	}
	if setting.WalletAddress != "" {
		setting.Config.Keys.WalletAddress = setting.WalletAddress
		_ = setting.FlushConfig()
	}
	getWalletPublicKey(ctx, filepath.Join(setting.Config.Home.AccountsPath, setting.WalletAddress+".json"), password)

	if created {
		pp.Log(ctx, "save the mnemonic phase properly for future recovery: \n"+
			"=======================================================================  \n"+
			mnemonic+"\n"+
			"======================================================================= \n")
		pp.Logf(ctx, "Wallet %v has been generated successfully", setting.WalletAddress)
	} else {
		pp.Logf(ctx, "Wallet %v already exists", setting.WalletAddress)
	}

	return setting.WalletAddress
}

func GetWalletAddress(ctx context.Context) error {
	files, err := os.ReadDir(setting.Config.Home.AccountsPath)

	if len(files) == 0 {
		return errors.New("account Dir is empty")
	}
	if err != nil {
		return err
	}

	walletAddress := setting.Config.Keys.WalletAddress

	password := setting.Config.Keys.WalletPassword
	fileName := walletAddress + ".json"

	for _, info := range files {
		if info.Name() == ".placeholder" || info.Name() != fileName {
			continue
		}
		utils.Log(info.Name())
		if getWalletPublicKey(ctx, filepath.Join(setting.Config.Home.AccountsPath, fileName), password) {
			setting.WalletAddress = walletAddress

			// get beneficiary address after get the wallet address
			err = getBeneficiaryAddress(walletAddress)
			if err != nil {
				return err
			}

			return nil
		}
		return errors.New("wrong password")
	}
	return errors.New("could not find the account file corresponds to the configured wallet address")
}

func getBeneficiaryAddress(walletAddressBech32 string) error {
	if setting.Config.Keys.BeneficiaryAddress == "" {
		setting.BeneficiaryAddress = walletAddressBech32
	} else {
		_, err := fwtypes.WalletAddressFromBech32(setting.Config.Keys.BeneficiaryAddress)
		if err != nil {
			return errors.New("invalid beneficiary address")
		}
		setting.BeneficiaryAddress = setting.Config.Keys.BeneficiaryAddress
	}
	return nil
}

func getWalletPublicKey(ctx context.Context, filePath, password string) bool {
	keyjson, err := os.ReadFile(filePath)
	if utils.CheckError(err) {
		pp.ErrorLog(ctx, "getWalletPublicKey ioutil.ReadFile", err)
		return false
	}
	key, err := fwtypes.DecryptKey(keyjson, password, true)

	if utils.CheckError(err) {
		pp.ErrorLog(ctx, "getWalletPublicKey DecryptKey", err)
		return false
	}
	setting.WalletPrivateKey = key.PrivateKey
	setting.WalletPublicKey = key.PrivateKey.PubKey()

	bech32PubKey, _ := fwtypes.WalletPubKeyToBech32(setting.WalletPublicKey)
	pp.DebugLog(ctx, "publicKey", bech32PubKey)
	pp.Log(ctx, "unlock wallet successfully ", setting.WalletAddress)
	return true
}

// Wallets get all wallets
func Wallets(ctx context.Context) []string {
	files, _ := os.ReadDir(setting.Config.Home.AccountsPath)
	var wallets []string
	for _, file := range files {
		fileName := file.Name()
		if fileName[len(fileName)-5:] == ".json" && fileName[:len(fwtypes.P2PAddressPrefix)] != fwtypes.P2PAddressPrefix {
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
