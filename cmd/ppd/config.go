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
	"github.com/stratosnet/stratos-chain/types"
)

const (
	createP2pKeyFlag = "create-p2p-key"
	createWalletFlag = "create-wallet"
)

func genConfig(cmd *cobra.Command, args []string) error {

	path, err := cmd.Flags().GetString(CONFIG)
	if err != nil {
		return errors.Wrap(err, "failed to get the configuration file path")
	}
	if path == defaultConfigPath {
		home, err := cmd.Flags().GetString(HOME)
		if err != nil {
			return err
		}
		path = filepath.Join(home, path)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(path), 0700)
	}
	err = setting.LoadConfig(path)
	if err != nil {
		fmt.Println("generating default config file")
		err = setting.GenDefaultConfig(path)
		if err != nil {
			return errors.Wrap(err, "failed to generate config file at given path")
		}

	}

	setting.LoadConfig(path)

	createP2pKey, err := cmd.Flags().GetBool(createP2pKeyFlag)
	if err == nil && createP2pKey {
		err = SetupP2PKey()
		if err != nil {
			err := errors.Wrap(err, "Couldn't setup PP node")
			utils.ErrorLog(err)
			return err
		}
	}

	createWallet, err := cmd.Flags().GetBool(createWalletFlag)
	if err == nil && createWallet {
		err = SetupWalletKey()
		if err != nil {
			utils.ErrorLog(err)
			return err
		}
	}
	return nil
}

func loadConfig(cmd *cobra.Command) error {
	homePath, err := cmd.Flags().GetString(HOME)
	if err != nil {
		utils.ErrorLog("failed to get 'home' path for the node")
		return err
	}
	homePath, err = utils.Absolute(homePath)
	if err != nil {
		return err
	}
	setting.SetupRoot(homePath)

	configPath, err := cmd.Flags().GetString(CONFIG)
	if err != nil {
		utils.ErrorLog("failed to get config path for the node")
		return err
	}
	if configPath == defaultConfigPath {
		configPath = filepath.Join(homePath, configPath)
	} else {
		configPath, err = utils.Absolute(configPath)
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat(configPath); err != nil {
		configPath = filepath.Join(homePath, configPath)
		if _, err := os.Stat(configPath); err != nil {
			return errors.Wrap(err, "not able to load config file, generate one with `ppd config`")
		}
	}

	setting.SetIPCEndpoint(homePath)

	err = setting.LoadConfig(configPath)
	if err != nil {
		return errors.Wrap(err, "failed to load config file")
	}

	if setting.Config.Debug {
		utils.MyLogger.SetLogLevel(utils.Debug)
	} else {
		utils.MyLogger.SetLogLevel(utils.Info)
	}

	if setting.Config.VersionShow != setting.Version {
		utils.ErrorLogf("config version and code version not match, config: [%s], code: [%s]", setting.Config.VersionShow, setting.Version)
	}

	return nil
}

func SetupWalletKey() error {
	if setting.Config.WalletAddress == "" {
		fmt.Println("No wallet key specified in config. Attempting to create one...")
		err := SetupWallet()
		if err != nil {
			utils.ErrorLog(err)
			return err
		}
	}
	return nil
}

func SetupWallet() error {
	nickname, err := console.Stdin.PromptInput("Enter wallet nickname: ")
	if err != nil {
		return errors.New("couldn't read nickname from console: " + err.Error())
	}
	password, err := console.Stdin.PromptPassword("Enter password: ")
	if err != nil {
		return errors.New("couldn't read password from console: " + err.Error())
	}
	confirmation, err := console.Stdin.PromptPassword("Enter password again: ")
	if err != nil {
		return errors.New("couldn't read confirmation password from console: " + err.Error())
	}
	if password != confirmation {
		return errors.New("invalid. The two passwords don't match")
	}

	mnemonic, err := console.Stdin.PromptPassword("input bip39 mnemonic (leave blank to generate a new one)")
	if mnemonic == "" {
		newMnemonic, err := utils.NewMnemonic()
		if err != nil {
			return errors.Wrap(err, "Couldn't generate new mnemonic")
		}
		mnemonic = newMnemonic
	}

	hdPath, err := console.Stdin.PromptInput("input hd-path for the account, default: \"m/44'/606'/0'/0/0\" : ")
	if err != nil {
		return errors.New("couldn't read the hd-path")
	}
	if hdPath == "" {
		hdPath = setting.HD_PATH
	}
	//hrp, mnemonic, bip39Passphrase, hdPath
	walletKeyAddress, err := utils.CreateWallet(setting.Config.AccountDir, nickname, password,
		types.StratosBech32Prefix, mnemonic, "", hdPath)
	if err != nil {
		return errors.New("couldn't create WalletAddress: " + err.Error())
	}

	walletKeyAddressString, err := walletKeyAddress.ToBech(types.StratosBech32Prefix)
	if err != nil {
		return errors.New("couldn't convert wallet address to bech string: " + err.Error())
	}

	fmt.Println("save the mnemonic phase properly for future recovery: \n" +
		"=======================================================================  \n" +
		mnemonic + "\n" +
		"======================================================================= \n")
	utils.Logf("Wallet %s has been generated successfully", walletKeyAddressString)

	save, err := console.Stdin.PromptInput("Do you want to use this wallet as your node wallet: Y(es)/N(o): ")
	if err != nil {
		return errors.New("couldn't read the input, not saving by default")
	}
	if strings.ToLower(save) == "yes" || strings.ToLower(save) == "y" {
		setting.SetConfig("WalletAddress", walletKeyAddressString)
		setting.SetConfig("WalletPassword", password)
	}

	return nil
}
