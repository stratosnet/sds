package main

import (
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/cmd/common"
	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp/setting"
)

const (
	addressFlag = "address"
)

func exportWallet(cmd *cobra.Command, _ []string) error {
	_, _, err := common.LoadConfig(cmd)
	if err != nil {
		return err
	}

	address, err := cmd.Flags().GetString(addressFlag)
	if err != nil {
		return err
	}
	if address == "" {
		address = setting.Config.Keys.WalletAddress
	}
	if address == "" {
		utils.Log("No wallet to export")
		return nil
	}

	password, err := cmd.Flags().GetString(passwordFlag)
	if err != nil {
		return err
	}

	walletJson, err := os.ReadFile(filepath.Join(setting.Config.Home.AccountsPath, address+".json"))
	if err != nil {
		return err
	}

	walletKey, err := fwtypes.DecryptKey(walletJson, password, true)
	if err != nil {
		return err
	}

	msg := "Wallet address: " + address
	pubKey, err := fwtypes.WalletPubKeyToBech32(walletKey.PrivateKey.PubKey())
	if err == nil {
		msg += "\nPublic key: " + pubKey
	}

	msg += "\nHdPath: " + walletKey.HdPath
	msg += "\nMnemonic: " + walletKey.Mnemonic

	utils.Log(msg)
	return nil
}

func exportP2pKey(cmd *cobra.Command, _ []string) error {
	_, _, err := common.LoadConfig(cmd)
	if err != nil {
		return err
	}

	address, err := cmd.Flags().GetString(addressFlag)
	if err != nil {
		return err
	}
	if address == "" {
		address = setting.Config.Keys.P2PAddress
	}
	if address == "" {
		utils.Log("No p2p key to export")
		return nil
	}

	password, err := cmd.Flags().GetString(passwordFlag)
	if err != nil {
		return err
	}

	p2pJson, err := os.ReadFile(filepath.Join(setting.Config.Home.AccountsPath, address+".json"))
	if err != nil {
		return err
	}

	p2pKey, err := fwtypes.DecryptKey(p2pJson, password, false)
	if err != nil {
		return err
	}

	msg := "P2P address: " + address
	pubKey, err := fwtypes.P2PPubKeyToBech32(p2pKey.PrivateKey.PubKey())
	if err == nil {
		msg += "\nPublic key: " + pubKey
	}

	if p2pKey.Mnemonic != "" {
		msg += "\nHdPath: " + p2pKey.HdPath
		msg += "\nMnemonic: " + p2pKey.Mnemonic
	} else {
		msg += "\nPrivate key: " + hex.EncodeToString(p2pKey.PrivateKey.Bytes())
	}

	utils.Log(msg)
	return nil
}
