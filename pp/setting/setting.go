package setting

import (
	ed25519crypto "crypto/ed25519"
	"errors"
	"fmt"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/relay/stratoschain/prefix"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/console"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// REPROTDHTIME 1 hour
const REPROTDHTIME = 60 * 60

// MAXDATA max slice size
const MAXDATA = 1024 * 1024 * 3

// HTTPTIMEOUT  HTTPTIMEOUT second
const HTTPTIMEOUT = 20

// FILEHASHLEN
const FILEHASHLEN = 64

// IMAGEPATH 保存图片路径
var IMAGEPATH = "./images/"

// ImageMap download image hash map
var ImageMap = &sync.Map{}

var VIDEOPATH = "./videos"

// DownProssMap download progress map
var DownProssMap = &sync.Map{}

// Config
var Config *config

// ConfigPath
var ConfigPath string

// IsLoad
var IsLoad bool

// UpLoadTaskIDMap
var UpLoadTaskIDMap = &sync.Map{}

// DownloadTaskIDMap
var DownloadTaskIDMap = &sync.Map{}

// socket map
var (
	UpMap     = make(map[string]interface{}, 0)
	DownMap   = make(map[string]interface{}, 0)
	ResultMap = make(map[string]interface{}, 0)
)

//  http code
var (
	FAILCode       = 500
	SUCCESSCode    = 0
	ShareErrorCode = 1002
	IsWindows      bool
)

type config struct {
	Version                     uint32
	VersionShow                 string
	DownloadPathMinLen          int
	Port                        string `yaml:"Port"`
	NetworkAddress              string `yaml:"NetworkAddress"`
	SPNetAddress                string `yaml:"SPNetAddress"`
	Debug                       bool   `yaml:"Debug"`
	PPListDir                   string `yaml:"PPListDir"`
	AccountDir                  string `yaml:"AccountDir"`
	ScryptN                     int    `yaml:"scryptN"`
	ScryptP                     int    `yaml:"scryptP"`
	DefPassword                 string `yaml:"DefPassword"`
	DefSavePath                 string `yaml:"DefSavePath"`
	StorehousePath              string `yaml:"StorehousePath"`
	DownloadPath                string `yaml:"DownloadPath"`
	P2PAddress                  string `yaml:"P2PAddress"`
	P2PPassword                 string `yaml:"P2PPassword"`
	WalletAddress               string `yaml:"WalletAddress"`
	WalletPassword              string `yaml:"WalletPassword"`
	AutoRun                     bool   `yaml:"AutoRun"`  // is auto login
	Internal                    bool   `yaml:"Internal"` // is internal net
	IsWallet                    bool   `yaml:"IsWallet"` // is wallet
	BPURL                       string `yaml:"BPURL"`    // bphttp
	IsCheckDefaultPath          bool   `yaml:"IsCheckDefaultPath"`
	IsLimitDownloadSpeed        bool   `yaml:"IsLimitDownloadSpeed"`
	LimitDownloadSpeed          uint64 `yaml:"LimitDownloadSpeed"`
	IsLimitUploadSpeed          bool   `yaml:"IsLimitUploadSpeed"`
	LimitUploadSpeed            uint64 `yaml:"LimitUploadSpeed"`
	IsCheckFileOperation        bool   `yaml:"IsCheckFileOperation"`
	IsCheckFileTransferFinished bool   `yaml:"IsCheckFileTransferFinished"`
	AddressPrefix               string `yaml:"AddressPrefix"`
	P2PKeyPrefix                string `yaml:"P2PKeyPrefix"`
	ChainId                     string `yaml:"ChainId"`
	Token                       string `yaml:"Token"`
	StratosChainAddress         string `yaml:"StratosChainAddress"`
	StratosChainPort            string `yaml:"StratosChainPort"`
	StreamingCache              bool   `yaml:"StreamingCache"`
}

var ostype = runtime.GOOS

// LoadConfig
func LoadConfig(configPath string) {
	ConfigPath = configPath
	Config = &config{}
	utils.LoadYamlConfig(Config, configPath)

	Config.Version = 5

	Config.VersionShow = "1.4"

	Config.DownloadPathMinLen = 112

	Config.ScryptN = 4096
	Config.ScryptP = 6
	if ostype == "windows" {
		IsWindows = true
		// imagePath = filepath.FromSlash(imagePath)
	} else {
		IsWindows = false
	}
	cf.SetLimitDownloadSpeed(Config.LimitDownloadSpeed, Config.IsLimitDownloadSpeed)
	cf.SetLimitUploadSpeed(Config.LimitUploadSpeed, Config.IsLimitUploadSpeed)
	prefix.SetConfig(Config.AddressPrefix)
}

// CheckLogin
func CheckLogin() bool {
	if WalletAddress == "" {
		utils.ErrorLog("please login")
		return false
	}
	return true
}

// GetSign
func GetSign(str string) []byte {
	sign := ed25519crypto.Sign(P2PPrivateKey, []byte(str))
	utils.DebugLog("GetSign == ", sign)
	return sign
}

// UpChan
var UpChan = make(chan string, 100)

// SetConfig SetConfig
func SetConfig(key, value string) bool {

	if !utils.CheckStructField(key, Config) {
		fmt.Println("configuration not found")
		return false
	}

	f, err := os.Open(ConfigPath)
	defer f.Close()

	if err != nil {
		fmt.Println("failed to change configuration file")
		return false
	}

	var contents []byte
	contents, err = ioutil.ReadAll(f)
	if err != nil {
		fmt.Println("failed to change configuration file")
		return false
	}

	contentStrs := strings.Split(string(contents), "\n")
	newString := ""
	change := false
	keyStr := key + ":"
	for _, str := range contentStrs {
		ss := strings.Split(str, " ")
		if len(ss) > 0 && ss[0] == keyStr {
			if keyStr == "DownloadPath:" {
				if ostype == "windows" {
					value = value + `\`
				} else {
					value = value + `/`
				}
			}
			ns := key + ": " + value
			newString += ns
			newString += "\n"
			change = true
			continue
		}
		newString += str
		newString += "\n"
	}
	if !change {
		return false
	}

	if os.Truncate(ConfigPath, 0) != nil {
		fmt.Println("failed to change configuration file")
		return false
	}

	var configOS *os.File
	configOS, err = os.OpenFile(ConfigPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	defer configOS.Close()

	if err != nil {
		fmt.Println("failed to change configuration file")
		return false
	}

	_, err = configOS.WriteString(newString)
	if err != nil {
		fmt.Println("failed to change configuration file")
		return false
	}

	LoadConfig(ConfigPath)
	fmt.Println("failed to change configuration file ", key+": ", value)
	return true
}

// SetupP2PKey Loads the existing P2P key for this node, or creates a new one if none is available.
func SetupP2PKey() error {
	if Config.P2PAddress == "" {
		fmt.Println("No P2P key specified in config. Attempting to create one...")
		nickname, err := console.Stdin.PromptInput("Enter P2pAddress nickname: ")
		if err != nil {
			return errors.New("couldn't read nickname from console: " + err.Error())
		}

		password, err := console.Stdin.PromptPassword("Enter password: ")
		if err != nil {
			return errors.New("couldn't read password from console: " + err.Error())
		}
		confimation, err := console.Stdin.PromptPassword("Enter password again: ")
		if err != nil {
			return errors.New("couldn't read confirmation password from console: " + err.Error())
		}
		if password != confimation {
			return errors.New("invalid. The two passwords don't match")
		}

		p2pKeyAddress, err := utils.CreateP2PKey(Config.AccountDir, nickname, password, Config.P2PKeyPrefix, Config.ScryptN, Config.ScryptP)
		if err != nil {
			return errors.New("couldn't create WalletAddress: " + err.Error())
		}

		p2pKeyAddressString, err := p2pKeyAddress.ToBech(Config.P2PKeyPrefix)
		if err != nil {
			return errors.New("couldn't convert P2P key address to bech string: " + err.Error())
		}
		Config.P2PAddress = p2pKeyAddressString
		Config.P2PPassword = password
		SetConfig("P2pAddress", p2pKeyAddressString)
		SetConfig("P2PPassword", password)
	}

	p2pKeyFile, err := ioutil.ReadFile(filepath.Join(Config.AccountDir, Config.P2PAddress+".json"))
	if err != nil {
		return errors.New("couldn't read P2P key file: " + err.Error())
	}

	p2pKey, err := utils.DecryptKey(p2pKeyFile, Config.P2PPassword)
	if err != nil {
		return errors.New("couldn't decrypt P2P key file: " + err.Error())
	}

	P2PAddress = Config.P2PAddress
	P2PPrivateKey = p2pKey.PrivateKey
	P2PPublicKey = ed25519.PrivKeyBytesToPubKeyBytes(P2PPrivateKey)
	return nil
}
