package common

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
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

	P2pMethodWallet = 0
	P2pMethodHex    = 1
	P2pMethodRandom = 2
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
		fmt.Println("failed to get 'home' path for the node")
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
	utils.Log("No p2p key specified in config. Attempting to create one...")

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

	var p2pKeyAddress fwcryptotypes.Address
	for len(p2pKeyAddress) == 0 {
		method, err := selectP2pCreationMethod()
		if err != nil {
			return errors.New("couldn't read p2p key creation method from console: " + err.Error())
		}
		switch method {
		case P2pMethodWallet:
			p2pKeyAddress, err = createP2pKeyFromWallet(nickname, password)
			if err != nil {
				utils.ErrorLogf("couldn't create p2p key from wallet: %v", err.Error())
				continue
			}

		case P2pMethodHex:
			p2pPrivKeyHex, err := console.Stdin.PromptInput("Enter p2p private key as a hex: ")
			if err != nil {
				utils.ErrorLogf("couldn't read p2p private key: %v", err.Error())
				continue
			}

			p2pKeyAddress, err = fwtypes.CreateP2PKey(setting.Config.Home.AccountsPath, nickname, password, p2pPrivKeyHex)
			if err != nil {
				return errors.New("couldn't create p2p key: " + err.Error())
			}

		case P2pMethodRandom:
			p2pKeyAddress, err = fwtypes.CreateP2PKey(setting.Config.Home.AccountsPath, nickname, password, "")
			if err != nil {
				return errors.New("couldn't create p2p key: " + err.Error())
			}
		}
	}

	p2pKeyAddressString, err := p2pKeyAddressToString(p2pKeyAddress)
	if err != nil {
		return err
	}

	setting.Config.Keys.P2PAddress = p2pKeyAddressString
	setting.Config.Keys.P2PPassword = password
	return setting.FlushConfig()
}

func selectP2pCreationMethod() (int, error) {
	prompt := fmt.Sprintf("How should the p2p key be generated?  " +
		"1) From the wallet  " +
		"2) From a hex-encoded private key  " +
		"3) Randomly: ")

	for {
		choiceStr, err := console.Stdin.PromptInput(prompt)
		if err != nil {
			return 0, err
		}
		choiceStr = strings.TrimSpace(choiceStr)
		if len(choiceStr) < 1 {
			continue
		}
		choice, err := strconv.ParseInt(choiceStr[0:1], 10, 64)
		if err != nil {
			return 0, err
		}

		switch choice {
		case 1:
			return P2pMethodWallet, nil
		case 2:
			return P2pMethodHex, nil
		case 3:
			return P2pMethodRandom, nil
		}
	}
}

func createP2pKeyFromWallet(nickname, password string) (fwcryptotypes.Address, error) {
	walletKey, err := loadWalletKey()
	if err != nil {
		return nil, err
	}

	confirmed := false
	hdPath := setting.HDPathP2p
	hdPathStart := hdPath[:strings.LastIndex(hdPath, "/")]
	hdPathEnd := hdPath[strings.LastIndex(hdPath, "/")+1:]
	for !confirmed {
		hdPath = hdPathStart + "/" + hdPathEnd
		p2pPrivateKey, err := fwtypes.GenerateP2pKeyFromHdPath(walletKey.Mnemonic, walletKey.Passphrase, hdPath)
		if err != nil {
			return nil, err
		}
		p2pKeyAddress := p2pPrivateKey.PubKey().Address()
		p2pKeyAddressString, err := p2pKeyAddressToString(p2pKeyAddress)
		if err != nil {
			return nil, err
		}
		confirmed, err = console.Stdin.PromptConfirm(fmt.Sprintf("Use the HD path (%v) to generate the p2p key (%v)?", hdPath, p2pKeyAddressString))
		if err != nil {
			return nil, err
		}
		if !confirmed {
			hdPathEnd, err = console.Stdin.PromptInput(fmt.Sprintf("What should the last number of the HD path be instead? (currently %v): ", hdPathEnd))
			if err != nil {
				return nil, err
			}
		}
	}

	p2pKeyAddress, created, err := fwtypes.CreateP2PKeyFromHdPath(setting.Config.Home.AccountsPath, nickname, password, walletKey.Mnemonic, walletKey.Passphrase, hdPath)
	if err != nil {
		return nil, err
	}
	if !created {
		if err = verifyP2pPassword(p2pKeyAddress, password); err != nil {
			return nil, errors.New("couldn't verify password of existing p2p key file: " + err.Error())
		}
	}

	return p2pKeyAddress, nil
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
	p2pKeyAddressString, err := p2pKeyAddressToString(p2pKeyAddress)
	if err != nil {
		return err
	}
	p2pKeyJson, err := os.ReadFile(filepath.Join(setting.Config.Home.AccountsPath, p2pKeyAddressString+".json"))
	if err != nil {
		return err
	}

	_, err = fwtypes.DecryptKey(p2pKeyJson, password, false)
	return err
}

func p2pKeyAddressToString(p2pKeyAddress fwcryptotypes.Address) (string, error) {
	p2pKeyAddressString := fwtypes.P2PAddressBytesToBech32(p2pKeyAddress.Bytes())
	if p2pKeyAddressString == "" {
		return "", errors.New("couldn't convert P2P key address to bech string")
	}
	return p2pKeyAddressString, nil
}

func NodePP(_ *cobra.Command, _ []string) error {
	err := RegisterDenoms()
	if err != nil {
		return errors.New(utils.FormatError(err))
	}

	debug.SetMemoryLimit(setting.SoftRamLimit)
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
