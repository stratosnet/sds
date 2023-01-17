package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
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
		//configPath = filepath.Join(homePath, configPath)
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

	if setting.Config.Version.Show != setting.Version {
		utils.ErrorLogf("config version and code version not match, config: [%s], code: [%s]", setting.Config.Version.Show, setting.Version)
	}

	return nil
}

func SetupWalletKey() error {
	if setting.Config.WalletAddress == "" {
		fmt.Println("No wallet key specified in config. Attempting to create one...")
		err := utils.SetupWallet(setting.Config.AccountDir, setting.HD_PATH, updateWalletConfig)
		if err != nil {
			utils.ErrorLog(err)
			return err
		}
	}
	return nil
}

func updateWalletConfig(walletKeyAddressString, password string) {
	setting.SetConfig("wallet_address", walletKeyAddressString)
	setting.SetConfig("wallet_password", password)
}
