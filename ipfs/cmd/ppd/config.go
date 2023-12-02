package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/stratosnet/sds/cmd/common"
	"github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp/setting"
)

const (
	createP2pKeyFlag = "create-p2p-key"
	createWalletFlag = "create-wallet"
)

func genConfig(cmd *cobra.Command, _ []string) error {
	path, err := cmd.Flags().GetString(common.Config)
	if err != nil {
		return errors.Wrap(err, "failed to get the configuration file path")
	}
	if path == common.DefaultConfigPath {
		home, err := cmd.Flags().GetString(common.Home)
		if err != nil {
			return err
		}
		path = filepath.Join(home, path)
	}

	if _, err = os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(path), 0700)
	}
	if err != nil {
		return err
	}

	err = setting.LoadConfig(path)
	if err != nil {
		fmt.Println("generating default config file")
		err = setting.GenDefaultConfig()
		if err != nil {
			return errors.Wrap(err, "failed to generate config file at given path")
		}
		if err = setting.LoadConfig(path); err != nil {
			return err
		}
	}

	createP2pKey, err := cmd.Flags().GetBool(createP2pKeyFlag)
	if err == nil && createP2pKey {
		err = common.SetupP2PKey()
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

func SetupWalletKey() error {
	if setting.Config.Keys.WalletAddress == "" {
		fmt.Println("No wallet key specified in config. Attempting to create one...")
		err := types.SetupWallet(setting.Config.Home.AccountsPath, setting.HDPath, updateWalletConfig)
		if err != nil {
			utils.ErrorLog(err)
			return err
		}
	}
	return nil
}

func updateWalletConfig(walletKeyAddressString, password string) {
	setting.Config.Keys.WalletAddress = walletKeyAddressString
	setting.Config.Keys.WalletPassword = password
	_ = setting.FlushConfig()
}
