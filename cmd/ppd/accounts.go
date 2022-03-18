package main

import (
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/console"
	"github.com/stratosnet/stratos-chain/types"
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
		p2pKeyAddress, err := utils.CreateP2PKey(setting.Config.AccountDir, "p2pkey", p2ppass,
			types.SdsNodeP2PAddressPrefix)
		if err != nil {
			return errors.New("couldn't create p2pkey: " + err.Error())
		}

		p2pKeyAddressString, err := p2pKeyAddress.ToBech(types.SdsNodeP2PAddressPrefix)
		if err != nil {
			return errors.New("couldn't convert P2P key address to bech string: " + err.Error())
		}
		setting.SetConfig("P2PAddress", p2pKeyAddressString)
		setting.SetConfig("P2PPassword", p2ppass)
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

	hdPath, err := cmd.Flags().GetString(hdPathFlag)
	if len(hdPath) <= 0 {
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
	setting.SetConfig("WalletAddress", walletKeyAddressString)

	save, _ := cmd.Flags().GetBool(savePassFlag)
	if save {
		setting.SetConfig("WalletPassword", password)
	}
	fmt.Println("save the mnemonic phase properly for future recover: \n" +
		"=======================================================================  \n" +
		mnemonic + "\n" +
		"======================================================================= \n")

	return nil
}

func findp2pKeyFiles() []string {
	files, _ := ioutil.ReadDir(setting.Config.AccountDir)
	var p2pkeyfiles []string
	for _, file := range files {
		fileName := file.Name()
		if fileName[len(fileName)-5:] == ".json" && fileName[:len(types.SdsNodeP2PAddressPrefix)] == types.SdsNodeP2PAddressPrefix {
			p2pkeyfiles = append(p2pkeyfiles, fileName)
		}
	}
	return p2pkeyfiles
}
