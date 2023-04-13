package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/pp/serv"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/console"
	utiltypes "github.com/stratosnet/sds/utils/types"
	"github.com/stratosnet/stratos-chain/types"
)

const (
	HOME              string = "home"
	CONFIG            string = "config"
	defaultConfigPath string = "./configs/config.toml"
)

var BaseServer = &serv.BaseServer{}

func nodePP(cmd *cobra.Command, args []string) error {
	registerDenoms()

	err := BaseServer.Start()
	defer BaseServer.Stop()
	if err != nil {
		return errors.New(utils.FormatError(err))
	}

	closure := make(chan os.Signal, 1)
	signal.Notify(closure,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)

	sig := <-closure
	utils.Logf("Quit signal detected: [%s]. Shutting down...", sig.String())
	return nil
}

func nodePreRunE(cmd *cobra.Command, args []string) error {
	err := loadConfig(cmd)
	if err != nil {
		return err
	}
	setting.MonitorInitialToken = serv.CreateInitialToken()
	setting.TrafficLogPath = filepath.Join(setting.GetRootPath(), "./tmp/logs/traffic_dump.log")
	trafficLogger := utils.NewTrafficLogger(setting.TrafficLogPath, false, true)
	trafficLogger.SetLogLevel(utils.Info)

	err = utils.InitIdWorker()
	if err != nil {
		return errors.Wrap(err, "Couldn't initialize id worker")
	}

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
		err = setting.FlushConfig()
		if err != nil {
			return err
		}
	}

	return nil
}

// RegisterDenoms registers the denominations to the PP.
func registerDenoms() {
	if err := utiltypes.RegisterDenom(utiltypes.Stos, sdktypes.OneDec()); err != nil {
		panic(err)
	}
	if err := utiltypes.RegisterDenom(utiltypes.Gwei, sdktypes.NewDecWithPrec(1, utiltypes.GweiDenomUnit)); err != nil {
		panic(err)
	}
	if err := utiltypes.RegisterDenom(utiltypes.Wei, sdktypes.NewDecWithPrec(1, utiltypes.WeiDenomUnit)); err != nil {
		panic(err)
	}
}
