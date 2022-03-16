package main

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/pp/api"
	"github.com/stratosnet/sds/pp/api/rest"
	"github.com/stratosnet/sds/pp/serv"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/console"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	"github.com/stratosnet/stratos-chain/types"
)

const (
	HOME              string = "home"
	CONFIG            string = "config"
	defaultConfigPath string = "./configs/config.yaml"
)

func nodePP(cmd *cobra.Command, args []string) error {
	if setting.Config.WalletAddress != "" && setting.Config.InternalPort != "" {
		go api.StartHTTPServ()
	}

	if setting.Config.RestPort != "" {
		go rest.StartHTTPServ()
	}

	serv.Start()
	return nil
}

func nodePreRunE(cmd *cobra.Command, args []string) error {
	err := loadConfig(cmd)
	if err != nil {
		return err
	}

	trafficLogger := utils.NewTrafficLogger(filepath.Join(setting.GetRootPath(), "./tmp/logs/traffic_dump.log"), false, true)
	trafficLogger.SetLogLevel(utils.Info)

	serv.StartDumpTrafficLog()
	err = SetupP2PKey()
	if err != nil {
		return errors.Wrap(err, "Couldn't setup PP node")
	}

	if _, err := os.Stat(setting.Config.PPListDir); os.IsNotExist(err) {
		if err = os.Mkdir(setting.Config.PPListDir, os.ModePerm); err != nil {
			return errors.Wrap(err, "Couldn't create PP list directory")
		}
	}
	return nil
}

// SetupP2PKey Loads the existing P2P key for this node, or creates a new one if none is available.
func SetupP2PKey() error {
	if setting.Config.P2PAddress == "" {
		utils.Log("No P2P key specified in config. Attempting to create one...")
		//nickname, err := console.Stdin.PromptInput("Enter P2PAddress nickname: ")
		//if err != nil {
		//	return errors.New("couldn't read nickname from console: " + err.Error())
		//}
		nickname := "p2pkey"
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

		p2pKeyAddress, err := utils.CreateP2PKey(setting.Config.AccountDir, nickname, password,
			types.SdsNodeP2PAddressPrefix)
		if err != nil {
			return errors.New("couldn't create p2p key: " + err.Error())
		}

		p2pKeyAddressString, err := p2pKeyAddress.ToBech(types.SdsNodeP2PAddressPrefix)
		if err != nil {
			return errors.New("couldn't convert P2P key address to bech string: " + err.Error())
		}
		setting.Config.P2PAddress = p2pKeyAddressString
		setting.Config.P2PPassword = password
		setting.SetConfig("P2PAddress", p2pKeyAddressString)
		setting.SetConfig("P2PPassword", password)
	}

	p2pKeyFile, err := ioutil.ReadFile(filepath.Join(setting.Config.AccountDir, setting.Config.P2PAddress+".json"))
	if err != nil {
		return errors.New("couldn't read P2P key file: " + err.Error())
	}

	p2pKey, err := utils.DecryptKey(p2pKeyFile, setting.Config.P2PPassword)
	if err != nil {
		return errors.New("couldn't decrypt P2P key file: " + err.Error())
	}

	setting.P2PAddress = setting.Config.P2PAddress
	setting.P2PPrivateKey = p2pKey.PrivateKey
	setting.P2PPublicKey = ed25519.PrivKeyBytesToPubKeyBytes(setting.P2PPrivateKey)
	return nil
}
