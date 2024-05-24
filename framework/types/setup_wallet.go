package types

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/framework/utils/console"
)

func SetupWallet(accountDir, defaultHDPath, bip39Passphrase string, updateConfig func(walletKeyAddressString, password string) error) error {
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
	if mnemonic == "" || err != nil {
		newMnemonic, err := NewMnemonic()
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
		hdPath = defaultHDPath
	}

	walletKeyAddress, created, err := CreateWallet(accountDir, nickname, password, mnemonic, bip39Passphrase, hdPath)
	if err != nil {
		return errors.New("couldn't create WalletAddress: " + err.Error())
	}

	walletKeyAddressString := WalletAddressBytesToBech32(walletKeyAddress.Bytes())
	if walletKeyAddressString == "" {
		return errors.New("couldn't convert wallet address to bech string")
	}

	if created {
		fmt.Println("save the mnemonic phase properly for future recovery: \n" +
			"=======================================================================  \n" +
			mnemonic + "\n" +
			"======================================================================= \n")
		utils.Logf("Wallet %v has been generated successfully", walletKeyAddressString)
	} else {
		utils.Logf("Wallet %v already exists", walletKeyAddressString)
	}

	save, err := console.Stdin.PromptInput("Do you want to use this wallet as your node wallet: Y(es)/N(o): ")
	if err != nil {
		return errors.New("couldn't read the input, not saving by default")
	}
	if strings.ToLower(save) == "yes" || strings.ToLower(save) == "y" {
		err = updateConfig(walletKeyAddressString, password)
	}

	return err
}
