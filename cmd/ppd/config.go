package main

import (
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/stratosnet/sds/cmd/common"
	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp/setting"
)

const (
	createP2pKeyFlag = "create-p2p-key"
	createWalletFlag = "create-wallet"
)

func genConfig(cmd *cobra.Command, _ []string) error {
	_, configPath, err := common.GetPaths(cmd, false)

	err = setting.LoadConfig(configPath)
	if err != nil {
		utils.Log("generating default config file")
		err = setting.GenDefaultConfig()
		if err != nil {
			return errors.Wrap(err, "failed to generate config file at given path")
		}
		if err = setting.LoadConfig(configPath); err != nil {
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
		err = setupWalletKey()
		if err != nil {
			utils.ErrorLog(err)
			return err
		}
	}
	return nil
}

func setupWalletKey() error {
	if setting.Config.Keys.WalletAddress != "" {
		return nil
	}

	utils.Log("No wallet key specified in config. Attempting to create one...")
	err := fwtypes.SetupWallet(setting.Config.Home.AccountsPath, setting.HDPath, updateWalletConfig)
	if err != nil {
		utils.ErrorLog(err)
		return err
	}
	return nil
}

func updateWalletConfig(walletKeyAddressString, password string) {
	setting.Config.Keys.WalletAddress = walletKeyAddressString
	setting.Config.Keys.WalletPassword = password
	_ = setting.FlushConfig()
}

func updateConfigVersion(cmd *cobra.Command, _ []string) error {
	_, configPath, err := common.LoadConfig(cmd)
	if err != nil {
		return err
	}

	// Load previous config
	prevVersion := setting.Config.Version.Show
	prevTree, err := toml.LoadFile(configPath)
	if err != nil {
		return err
	}
	prevTreeFlat := flattenTomlTree(prevTree)

	// Update config
	setting.Config.Version = setting.VersionConfig{
		AppVer:    setting.AppVersion,
		MinAppVer: setting.MinAppVersion,
		Show:      setting.Version,
	}
	if err = setting.FlushConfig(); err != nil {
		return err
	}
	curTree, err := toml.LoadFile(configPath)
	if err != nil {
		return err
	}
	curTreeFlat := flattenTomlTree(curTree)

	// Identify config changes
	if setting.Config.Version.Show != prevVersion {
		utils.Logf("Updated config version from %v to %v", prevVersion, setting.Config.Version.Show)
	}

	for key, value := range prevTreeFlat {
		if _, found := curTreeFlat[key]; !found {
			utils.Logf("Deleted entry %v = %v", key, value)
		}
	}
	for key, value := range curTreeFlat {
		if _, found := prevTreeFlat[key]; !found {
			utils.Logf("Added entry %v = %v", key, value)
		}
	}
	return nil
}

func flattenTomlTree(root *toml.Tree) map[string]any {
	flattenedTree := make(map[string]any)

	var helper func(*toml.Tree, string)
	helper = func(tree *toml.Tree, prefix string) {
		for key, value := range tree.Values() {
			fullKey := key
			if prefix != "" {
				fullKey = prefix + "." + key
			}

			if subtree, ok := value.(*toml.Tree); ok {
				helper(subtree, fullKey)
			} else {
				if tomlVal, ok := value.(*toml.PubTOMLValue); ok {
					flattenedTree[fullKey] = tomlVal.Value()
				} else {
					flattenedTree[fullKey] = value
				}
			}
		}
	}

	helper(root, "")
	return flattenedTree
}
