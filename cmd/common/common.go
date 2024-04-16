package common

import (
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"

	sdkmath "cosmossdk.io/math"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	fwcryptotypes "github.com/stratosnet/sds/framework/crypto/types"

	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/framework/utils/console"
	"github.com/stratosnet/sds/pp/serv"
	"github.com/stratosnet/sds/pp/setting"
	txclienttypes "github.com/stratosnet/sds/tx-client/types"
)

const (
	Home              string = "home"
	Config            string = "config"
	DefaultConfigPath string = "./config/config.toml"
)

var BaseServer = &serv.BaseServer{}

func GetQuitChannel() chan os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)

	return quit
}

func RootPreRunE(cmd *cobra.Command, _ []string) error {
	homePath, err := cmd.Flags().GetString(Home)
	if err != nil {
		utils.ErrorLog("failed to get 'home' path for the node")
		return err
	}
	homePath, err = utils.Absolute(homePath)
	if err != nil {
		return err
	}
	setting.SetupRoot(homePath)
	utils.NewDefaultLogger(filepath.Join(setting.GetRootPath(), "./tmp/logs/stdout.log"), true, true)
	return nil
}

func NodePreRunE(cmd *cobra.Command, _ []string) error {
	_, _, err := LoadConfig(cmd)
	if err != nil {
		return err
	}

	err = setting.InitializeSPMap()
	if err != nil {
		return err
	}

	setting.MonitorInitialToken = serv.CreateInitialToken()
	setting.TrafficLogPath = filepath.Join(setting.GetRootPath(), "./tmp/logs/traffic_dump.log")
	trafficLogger := utils.NewTrafficLogger(setting.TrafficLogPath, false, true)
	trafficLogger.SetLogLevel(utils.Info)

	err = SetupP2PKey()
	if err != nil {
		return errors.Wrap(err, "Couldn't setup PP node")
	}

	if _, err := os.Stat(setting.Config.Home.PeersPath); os.IsNotExist(err) {
		if err = os.Mkdir(setting.Config.Home.PeersPath, os.ModePerm); err != nil {
			return errors.Wrap(err, "Couldn't create PP list directory")
		}
	}
	return nil
}

func LoadConfig(cmd *cobra.Command) (homePath, configPath string, err error) {
	homePath, configPath, err = GetPaths(cmd, true)
	if err != nil {
		utils.ErrorLog("failed to get 'home' path for the node")
		return
	}

	setting.SetIPCEndpoint(homePath)

	err = setting.LoadConfig(configPath)
	if err != nil {
		err = errors.Wrap(err, "failed to load config file")
		return
	}

	if setting.Config.Node.Debug {
		utils.MyLogger.SetLogLevel(utils.Debug)
	} else {
		utils.MyLogger.SetLogLevel(utils.Info)
	}

	if setting.Config.Version.Show != setting.Version {
		utils.ErrorLogf("config version and code version not match, config: [%s], code: [%s]", setting.Config.Version.Show, setting.Version)
	}

	return
}

func GetPaths(cmd *cobra.Command, errOnMissingDir bool) (homePath, configPath string, err error) {
	homePath, err = cmd.Flags().GetString(Home)
	if err != nil {
		utils.ErrorLog("failed to get 'home' path for the node")
		return
	}
	homePath, err = utils.Absolute(homePath)
	if err != nil {
		return
	}
	setting.SetupRoot(homePath)

	configPath, err = cmd.Flags().GetString(Config)
	if err != nil {
		utils.ErrorLog("failed to get config path for the node")
		return
	}

	if configPath == DefaultConfigPath {
		configPath = filepath.Join(homePath, configPath)
	} else {
		configPath, err = utils.Absolute(configPath)
		if err != nil {
			return
		}
	}

	if _, err = os.Stat(configPath); err != nil {
		if os.IsNotExist(err) && !errOnMissingDir {
			err = os.MkdirAll(filepath.Dir(configPath), 0700)
		}
	}
	return
}

// SetupP2PKey Loads the existing P2P key for this node, or creates a new one if none is available.
func SetupP2PKey() error {
	if setting.Config.Keys.P2PAddress != "" {
		return nil
	}
	utils.Log("No P2P key specified in config. Attempting to create one...")

	nickname := "p2pkey"
	password, err := console.Stdin.PromptPassword("Enter password for p2p key: ")
	if err != nil {
		return errors.New("couldn't read password from console: " + err.Error())
	}
	confirmation, err := console.Stdin.PromptPassword("Enter password for p2p key again: ")
	if err != nil {
		return errors.New("couldn't read confirmation password from console: " + err.Error())
	}
	if password != confirmation {
		return errors.New("invalid. The two passwords don't match")
	}

	p2pPrivKeyHex, err := console.Stdin.PromptInput("Enter p2p private key as a hex (leave blank to generate a new one): ")
	if err != nil {
		return errors.New("couldn't read p2p private key: " + err.Error())
	}

	var p2pKeyAddress fwcryptotypes.Address
	walletKey, err := loadWalletKey()
	if err != nil {
		utils.ErrorLogf("couldn't load wallet info: %v", err.Error())
	}
	if err == nil && p2pPrivKeyHex == "" {
		var created bool
		p2pKeyAddress, created, err = fwtypes.CreateP2PKeyFromHdPath(setting.Config.Home.AccountsPath, nickname, password, walletKey.Mnemonic, walletKey.Passphrase, walletKey.HdPath)
		if err != nil {
			return errors.New("couldn't create p2p key from wallet: " + err.Error())
		}
		if !created {
			if err = verifyP2pPassword(p2pKeyAddress, password); err != nil {
				return errors.New("couldn't verify password of existing p2p key file: " + err.Error())
			}
		}
	}

	if len(p2pKeyAddress) == 0 {
		if p2pPrivKeyHex == "" {
			utils.Logf("p2p key will be randomly generated")
		}

		p2pKeyAddress, err = fwtypes.CreateP2PKey(setting.Config.Home.AccountsPath, nickname, password, p2pPrivKeyHex)
		if err != nil {
			return errors.New("couldn't create p2p key: " + err.Error())
		}
	}

	p2pKeyAddressString := fwtypes.P2PAddressBytesToBech32(p2pKeyAddress.Bytes())
	if p2pKeyAddressString == "" {
		return errors.New("couldn't convert P2P key address to bech string: ")
	}

	setting.Config.Keys.P2PAddress = p2pKeyAddressString
	setting.Config.Keys.P2PPassword = password
	return setting.FlushConfig()
}

func loadWalletKey() (*fwtypes.AccountKey, error) {
	if setting.Config.Keys.WalletAddress == "" {
		return nil, errors.New("no wallet specified in the config file")
	}

	walletJson, err := os.ReadFile(filepath.Join(setting.Config.Home.AccountsPath, setting.Config.Keys.WalletAddress+".json"))
	if err != nil {
		return nil, err
	}

	password := setting.Config.Keys.WalletPassword
	if password == "" {
		password, err = console.Stdin.PromptPassword("Enter wallet password: ")
		if err != nil {
			return nil, errors.New("couldn't read wallet password from console: " + err.Error())
		}
	}
	return fwtypes.DecryptKey(walletJson, password, true)
}

func verifyP2pPassword(p2pKeyAddress fwcryptotypes.Address, password string) error {
	p2pKeyAddressString := fwtypes.P2PAddressBytesToBech32(p2pKeyAddress.Bytes())
	p2pKeyJson, err := os.ReadFile(filepath.Join(setting.Config.Home.AccountsPath, p2pKeyAddressString+".json"))
	if err != nil {
		return err
	}

	_, err = fwtypes.DecryptKey(p2pKeyJson, password, false)
	return err
}

func NodePP(_ *cobra.Command, _ []string) error {
	err := RegisterDenoms()
	if err != nil {
		return errors.New(utils.FormatError(err))
	}

	debug.SetMemoryLimit(setting.SoftRamLimitTier2)
	err = BaseServer.Start()
	defer BaseServer.Stop()
	if err != nil {
		return errors.New(utils.FormatError(err))
	}

	closure := GetQuitChannel()
	sig := <-closure
	utils.Logf("Quit signal detected: [%s]. Shutting down...", sig.String())
	return nil
}

// RegisterDenoms registers the denominations to the PP.
func RegisterDenoms() error {
	if err := txclienttypes.RegisterDenom(txclienttypes.Stos, sdkmath.LegacyOneDec()); err != nil {
		return err
	}
	if err := txclienttypes.RegisterDenom(txclienttypes.Gwei, sdkmath.LegacyNewDecWithPrec(1, txclienttypes.GweiDenomUnit)); err != nil {
		return err
	}
	if err := txclienttypes.RegisterDenom(txclienttypes.Wei, sdkmath.LegacyNewDecWithPrec(1, txclienttypes.WeiDenomUnit)); err != nil {
		return err
	}

	return nil
}
