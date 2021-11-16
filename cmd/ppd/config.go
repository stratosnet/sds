package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/console"
)

func genConfig(cmd *cobra.Command, args []string) error {

	path, err := cmd.Flags().GetString(CONFIG)
	if err != nil {
		return errors.Wrap(err, "failed to get the configuration file path")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(path), 0700)
	}
	err = setting.LoadConfig(path)
	if err != nil {
		err = setting.GenDefaultConfig(path)
		if err != nil {
			return errors.Wrap(err, "failed to generate config file at given path")
		}

	}

	setting.LoadConfig(path)

	err = SetupP2PKey()
	if err != nil {
		err := errors.Wrap(err, "Couldn't setup PP node")
		utils.ErrorLog(err)
		return err
	}

	err = SetupWalletKey()
	if err != nil {
		utils.ErrorLog(err)
	}
	return nil
}

func SetupWalletKey() error {
	if setting.Config.WalletAddress == "" {
		fmt.Println("No wallet key specified in config. Attempting to create one...")
		nickname, err := console.Stdin.PromptInput("Enter wallet nickname: ")
		if err != nil {
			return errors.New("couldn't read nickname from console: " + err.Error())
		}

		password, err := console.Stdin.PromptPassword("Enter password: ")
		if err != nil {
			return errors.New("couldn't read password from console: " + err.Error())
		}
		confimation, err := console.Stdin.PromptPassword("Enter password again: ")
		if err != nil {
			return errors.New("couldn't read confirmation password from console: " + err.Error())
		}
		if password != confimation {
			return errors.New("invalid. The two passwords don't match")
		}

		mnemonic := console.MyGetPassword("input bip39 mnemonic (leave blank to generate a new one)", false)
		if mnemonic == "" {
			newMnemonic, err := utils.NewMnemonic()
			if err != nil {
				return errors.Wrap(err, "Couldn't generate new mnemonic")
			}
			mnemonic = newMnemonic
			fmt.Println("generated mnemonic is :  \n" +
				"=======================================================================  \n" +
				mnemonic + "\n" +
				"======================================================================= \n")
		}

		hdPath, err := console.Stdin.PromptInput("input hd-path for the account, default: m/44'/606'/0'/0/0 ")
		if err != nil {
			return errors.New("couldn't read the hd-path")
		}
		if hdPath == "" {
			hdPath = setting.HD_PATH
		}
		//hrp, mnemonic, bip39Passphrase, hdPath
		walletKeyAddress, err := utils.CreateWallet(setting.Config.AccountDir, nickname, password,
			setting.Config.AddressPrefix, mnemonic, "", hdPath, setting.Config.ScryptN, setting.Config.ScryptP)
		if err != nil {
			return errors.New("couldn't create WalletAddress: " + err.Error())
		}

		walletKeyAddressString, err := walletKeyAddress.ToBech(setting.Config.AddressPrefix)
		if err != nil {
			return errors.New("couldn't convert P2P key address to bech string: " + err.Error())
		}
		setting.SetConfig("WalletAddress", walletKeyAddressString)

		save, err := console.Stdin.PromptInput("save wallet password to config file: Y(es)/N(o)")
		if err != nil {
			return errors.New("couldn't read the input, not saving by default")
		}
		if strings.ToLower(save) == "yes" || strings.ToLower(save) == "y" {
			setting.SetConfig("WalletPassword", password)
		}
		fmt.Println("save the mnemonic phase properly for future recover: \n" +
			"=======================================================================  \n" +
			mnemonic + "\n" +
			"======================================================================= \n")
	}

	return nil
}
