package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/utils/console"

	"github.com/stratosnet/sds/pp/setting"
)

const (
	hdPathFlag   = "hd-path"
	mnemonicFlag = "mnemonic"
	passwordFlag = "password"
	savePassFlag = "save-pass"
	nicknameFlag = "nickname"

	p2pPassFlag   = "p2p-pass"
	newP2pKeyFlag = "new-p2p-key"
)

func createAccounts(cmd *cobra.Command, args []string) error {
	p2ppass, _ := cmd.Flags().GetString(p2pPassFlag)

	newP2pKey, _ := cmd.Flags().GetBool(newP2pKeyFlag)
	p2pkeyfiles := findp2pKeyFiles()

	if len(p2pkeyfiles) < 1 || newP2pKey {
		fmt.Println("generating new p2p key")
		p2pKeyAddress, err := fwtypes.CreateP2PKey(setting.Config.Home.AccountsPath, "p2pkey", p2ppass)
		if err != nil {
			return errors.New("couldn't create p2pkey: " + err.Error())
		}

		p2pKeyAddressString := fwtypes.P2PAddressBytesToBech32(p2pKeyAddress.Bytes())
		if p2pKeyAddressString == "" {
			return errors.New("couldn't convert P2P key address to bech string")
		}
		setting.Config.Keys.P2PAddress = p2pKeyAddressString
		setting.Config.Keys.P2PPassword = p2ppass
		err = setting.FlushConfig()
		if err != nil {
			return err
		}
	}

	nickname, _ := cmd.Flags().GetString(nicknameFlag)
	if len(nickname) <= 0 {
		nickname = "wallet"
	}

	password, _ := cmd.Flags().GetString(passwordFlag)
	if len(password) <= 0 {
		newPassword, err := console.Stdin.PromptPassword("Enter wallet password: ")
		if err != nil {
			return errors.New("couldn't read password from input: " + err.Error())
		}
		password = newPassword
	}

	mnemonic, _ := cmd.Flags().GetString(mnemonicFlag)
	if len(mnemonic) <= 0 {
		newMnemonic, err := fwtypes.NewMnemonic()
		if err != nil {
			return errors.Wrap(err, "Couldn't generate new mnemonic")
		}
		mnemonic = newMnemonic
		fmt.Println("generated mnemonic is :  \n" +
			"=======================================================================  \n" +
			mnemonic + "\n" +
			"======================================================================= \n")
	}

	hdPath, err := cmd.Flags().GetString(hdPathFlag)
	if err != nil {
		return err
	}
	if len(hdPath) <= 0 {
		hdPath = setting.HDPath
	}
	//hrp, mnemonic, bip39Passphrase, hdPath
	walletKeyAddress, err := fwtypes.CreateWallet(setting.Config.Home.AccountsPath, nickname, password, mnemonic, "", hdPath)
	if err != nil {
		return errors.New("couldn't create WalletAddress: " + err.Error())
	}

	walletKeyAddressString := fwtypes.WalletAddressBytesToBech32(walletKeyAddress.Bytes()) //walletKeyAddress.ToBech(fwtypes.StratosBech32Prefix)
	if walletKeyAddressString == "" {
		return errors.New("couldn't convert wallet address to bech string")
	}
	setting.Config.Keys.WalletAddress = walletKeyAddressString

	save, _ := cmd.Flags().GetBool(savePassFlag)
	if save {
		setting.Config.Keys.WalletPassword = password
	}

	err = setting.FlushConfig()
	if err != nil {
		return err
	}

	fmt.Println("save the mnemonic phase properly for future recover: \n" +
		"=======================================================================  \n" +
		mnemonic + "\n" +
		"======================================================================= \n")

	return nil
}

func findp2pKeyFiles() []string {
	files, _ := os.ReadDir(setting.Config.Home.AccountsPath)
	var p2pkeyfiles []string
	for _, file := range files {
		fileName := file.Name()
		if fileName[len(fileName)-5:] == ".json" && fileName[:len(fwtypes.P2PAddressPrefix)] == fwtypes.P2PAddressPrefix {
			p2pkeyfiles = append(p2pkeyfiles, fileName)
		}
	}
	return p2pkeyfiles
}
