package utils

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stratosnet/framework/utils/console"
	"github.com/stratosnet/tx-client/types"
)

func SetupWallet(accountDir, defaultHDPath string, updateConfig func(walletKeyAddressString, password string)) error {
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
	//hrp, mnemonic, bip39Passphrase, hdPath
	walletKeyAddress, err := CreateWallet(accountDir, nickname, password,
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
	Logf("Wallet %s has been generated successfully", walletKeyAddressString)

	save, err := console.Stdin.PromptInput("Do you want to use this wallet as your node wallet: Y(es)/N(o): ")
	if err != nil {
		return errors.New("couldn't read the input, not saving by default")
	}
	if strings.ToLower(save) == "yes" || strings.ToLower(save) == "y" {
		updateConfig(walletKeyAddressString, password)
	}

	return nil
}
