package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	fwcryptotypes "github.com/stratosnet/sds/framework/crypto/types"
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

	p2pPassFlag    = "p2p-pass"
	newP2pKeyFlag  = "new-p2p-key"
	hdPathP2pFlag  = "hd-path-p2p"
	p2pPrivKeyFlag = "p2p-priv-key"
)

func createAccounts(cmd *cobra.Command, _ []string) error {
	// Parse flag values
	p2pPassword, _ := cmd.Flags().GetString(p2pPassFlag)
	newP2pKey, _ := cmd.Flags().GetBool(newP2pKeyFlag)

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

	save, _ := cmd.Flags().GetBool(savePassFlag)
	if save {
		setting.Config.Keys.WalletPassword = password
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

	hdPathP2p, err := cmd.Flags().GetString(hdPathP2pFlag)
	if err != nil {
		return err
	}
	if len(hdPathP2p) <= 0 {
		hdPathP2p = setting.HDPathP2p
	}

	p2pPrivateKey, err := cmd.Flags().GetString(p2pPrivKeyFlag)
	if err != nil {
		return err
	}

	// Create accounts
	newWalletAddress, err := createWallet(nickname, password, mnemonic, hdPath)
	if err != nil {
		return err
	}
	if newWalletAddress != "" {
		fmt.Println("Created wallet " + newWalletAddress)
	}

	newP2pAddress, err := createP2pKey(p2pPassword, mnemonic, hdPathP2p, p2pPrivateKey, newP2pKey)
	if err != nil {
		return err
	}
	if newP2pAddress != "" {
		fmt.Println("Created p2p key " + newP2pAddress)
	}

	if newWalletAddress != "" || newP2pAddress != "" {
		fmt.Println("Make sure to save the mnemonic phase properly! It's the only way to recover your wallet or p2p key if you lose the .json file: \n" +
			"=======================================================================  \n" +
			mnemonic + "\n" +
			"======================================================================= \n")
		err = setting.FlushConfig()
		if err != nil {
			return err
		}
	}

	return nil
}

func createWallet(nickname, password, mnemonic, hdPath string) (string, error) {
	walletKeyAddress, created, err := fwtypes.CreateWallet(setting.Config.Home.AccountsPath, nickname, password, mnemonic, setting.Bip39Passphrase, hdPath)
	if err != nil {
		return "", errors.New("couldn't create wallet: " + err.Error())
	}
	if !created {
		fmt.Println("Wallet already exists")
		return "", nil
	}

	walletKeyAddressString := fwtypes.WalletAddressBytesToBech32(walletKeyAddress.Bytes())
	if walletKeyAddressString == "" {
		return "", errors.New("couldn't convert wallet address to bech string")
	}

	setting.Config.Keys.WalletAddress = walletKeyAddressString
	return walletKeyAddressString, nil
}

func createP2pKey(password, mnemonic, hdPath, privateKey string, newP2pKey bool) (string, error) {
	if len(findP2pKeyFiles()) > 0 && !newP2pKey {
		fmt.Println("there is already a p2p key")
		return "", nil
	}

	var p2pKeyAddress fwcryptotypes.Address
	var err error
	if privateKey != "" {
		// Recreate key from given private key
		p2pKeyAddress, err = fwtypes.CreateP2PKey(setting.Config.Home.AccountsPath, "p2pkey", password, privateKey)
		if err != nil {
			return "", errors.New("couldn't recreate p2p key from given private key: " + err.Error())
		}
	}
	if len(p2pKeyAddress) == 0 {
		var created bool
		p2pKeyAddress, created, err = fwtypes.CreateP2PKeyFromHdPath(setting.Config.Home.AccountsPath, "p2pkey", password, mnemonic, setting.Bip39Passphrase, hdPath)
		if !created && newP2pKey {
			p2pKeyAddress = nil
			err = nil
		}
	}
	if len(p2pKeyAddress) == 0 && err == nil {
		fmt.Println("p2p key will be randomly generated")
		p2pKeyAddress, err = fwtypes.CreateP2PKey(setting.Config.Home.AccountsPath, "p2pkey", password, "")
	}
	if err != nil {
		return "", errors.New("couldn't create p2p key: " + err.Error())
	}

	p2pKeyAddressString := fwtypes.P2PAddressBytesToBech32(p2pKeyAddress.Bytes())
	if p2pKeyAddressString == "" {
		return "", errors.New("couldn't convert p2p key address to bech string")
	}

	setting.Config.Keys.P2PAddress = p2pKeyAddressString
	setting.Config.Keys.P2PPassword = password
	return p2pKeyAddressString, nil
}

func findP2pKeyFiles() []string {
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
