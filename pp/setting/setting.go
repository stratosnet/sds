package setting

import (
	ed25519crypto "crypto/ed25519"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/relay/stratoschain"
	"github.com/stratosnet/sds/utils"
)

const (
	Version = "v0.6.0"
	HD_PATH = "m/44'/606'/0'/0/0"

	// REPORTDHTIME 1 hour
	REPORTDHTIME = 60 * 60
	// NodeReportIntervalSec Interval of node stat report
	NodeReportIntervalSec   = 300    // in seconds
	NodeReportCheckInterval = 1000   // in num of heights
	WeightDeductionInterval = 100000 // interval for weight deduction in heights

	// MAXDATA max slice size
	MAXDATA = 1024 * 1024 * 3
	// HTTPTIMEOUT  HTTPTIMEOUT second
	HTTPTIMEOUT = 20
	// IMAGEPATH
	IMAGEPATH = "./images/"
	VIDEOPATH = "./videos"

	STREAM_CACHE_MAXSLICE = 2

	FILE_SLICE_DOWNLOAD_BATCH_SIZE = 20
)

var (
	// Config
	Config *config
	// ConfigPath
	ConfigPath string
	rootPath   string

	// ImageMap download image hash map
	ImageMap = &sync.Map{}

	// DownloadProgressMap download progress map
	DownloadProgressMap = &sync.Map{}

	// IsLoad
	IsLoad bool

	// UploadTaskIDMap
	UploadTaskIDMap = &sync.Map{}

	// DownloadTaskIDMap
	DownloadTaskIDMap = &sync.Map{}

	// socket map
	UpMap     = make(map[string]interface{}, 0)
	DownMap   = make(map[string]interface{}, 0)
	ResultMap = make(map[string]interface{}, 0)

	//  http code
	FAILCode       = 500
	SUCCESSCode    = 0
	ShareErrorCode = 1002

	ostype    = runtime.GOOS
	IsWindows bool

	// UpChan
	UpChan = make(chan string, 100)
)

type SPBaseInfo struct {
	P2PAddress     string `yaml:"P2PAddress"`
	P2PPublicKey   string `yaml:"P2PPublicKey"`
	NetworkAddress string `yaml:"NetworkAddress"`
}

type config struct {
	Version              uint32
	VersionShow          string
	DownloadPathMinLen   int
	Port                 string       `yaml:"Port"`
	NetworkAddress       string       `yaml:"NetworkAddress"`
	Debug                bool         `yaml:"Debug"`
	PPListDir            string       `yaml:"PPListDir"`
	AccountDir           string       `yaml:"AccountDir"`
	StorehousePath       string       `yaml:"StorehousePath"`
	DownloadPath         string       `yaml:"DownloadPath"`
	P2PAddress           string       `yaml:"P2PAddress"`
	P2PPassword          string       `yaml:"P2PPassword"`
	WalletAddress        string       `yaml:"WalletAddress"`
	WalletPassword       string       `yaml:"WalletPassword"`
	AutoRun              bool         `yaml:"AutoRun"`  // is auto login
	Internal             bool         `yaml:"Internal"` // is internal net
	IsWallet             bool         `yaml:"IsWallet"` // is wallet
	IsLimitDownloadSpeed bool         `yaml:"IsLimitDownloadSpeed"`
	LimitDownloadSpeed   uint64       `yaml:"LimitDownloadSpeed"`
	IsLimitUploadSpeed   bool         `yaml:"IsLimitUploadSpeed"`
	LimitUploadSpeed     uint64       `yaml:"LimitUploadSpeed"`
	ChainId              string       `yaml:"ChainId"`
	Token                string       `yaml:"Token"`
	StratosChainUrl      string       `yaml:"StratosChainUrl"`
	RestPort             string       `yaml:"RestPort"`
	InternalPort         string       `yaml:"InternalPort"`
	TrafficLogInterval   uint64       `yaml:"TrafficLogInterval"`
	SPList               []SPBaseInfo `yaml:"SPList"`
}

func SetupRoot(root string) {
	rootPath = root
}

func GetRootPath() string {
	return rootPath
}

// LoadConfig
func LoadConfig(configPath string) error {
	ConfigPath = configPath
	Config = &config{}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		utils.Log("The config at location", configPath, "does not exist")
		return err
	}
	err := utils.LoadYamlConfig(Config, configPath)
	if err != nil {
		return err
	}

	Config.DownloadPathMinLen = 88

	err = formalizePath(Config)
	if err != nil {
		return err
	}

	if ostype == "windows" {
		IsWindows = true
		// imagePath = filepath.FromSlash(imagePath)
	} else {
		IsWindows = false
	}

	cf.SetLimitDownloadSpeed(Config.LimitDownloadSpeed, Config.IsLimitDownloadSpeed)
	cf.SetLimitUploadSpeed(Config.LimitUploadSpeed, Config.IsLimitUploadSpeed)
	IsAuto = Config.AutoRun
	utils.DebugLogf("AutoRun flag: %v", IsAuto)
	// todo: we shouldn't call stratoschain package to setup a global variable
	stratoschain.Url = Config.StratosChainUrl

	// Initialize SPMap
	for _, sp := range Config.SPList {
		key := sp.P2PAddress
		if key == "" {
			key = "unknown"
		}
		SPMap.Store(key, sp)
	}

	return nil
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
	utils.DebugLog("GetSign == ", hex.EncodeToString(sign))
	return sign
}

// SetConfig SetConfig
func SetConfig(key, value string) bool {

	if !utils.CheckStructField(key, Config) {
		utils.Log("configuration not found")
		return false
	}

	f, err := os.Open(ConfigPath)
	defer f.Close()
	if err != nil {
		utils.ErrorLog("failed to change configuration file", err)
		return false
	}

	var contents []byte
	contents, err = ioutil.ReadAll(f)
	if err != nil {
		utils.ErrorLog("failed to read configuration file", err)
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

	if err = os.Truncate(ConfigPath, 0); err != nil {
		utils.ErrorLog("failed to change configuration file", err)
		return false
	}

	var configOS *os.File
	configOS, err = os.OpenFile(ConfigPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	defer configOS.Close()

	if err != nil {
		utils.ErrorLog("failed to change configuration file", err)
		return false
	}

	_, err = configOS.WriteString(newString)
	if err != nil {
		utils.ErrorLog("failed to change configuration file", err)
		return false
	}

	LoadConfig(ConfigPath)
	if !(strings.Contains(strings.ToLower(key), "password") || strings.Contains(strings.ToLower(key), "pw")) {
		utils.Log("finished changing configuration file ", key+": ", value)
	}
	//prefix.SetConfig(Config.AddressPrefix)

	return true
}

func defaultConfig() *config {
	return &config{
		Version:              6,
		VersionShow:          Version,
		DownloadPathMinLen:   0,
		Port:                 "18081",
		NetworkAddress:       "127.0.0.1",
		Debug:                false,
		PPListDir:            "./peers",
		AccountDir:           "./accounts",
		StorehousePath:       "./storage",
		DownloadPath:         "./download",
		P2PAddress:           "",
		P2PPassword:          "",
		WalletAddress:        "",
		WalletPassword:       "",
		AutoRun:              true,
		Internal:             false,
		IsWallet:             true,
		IsLimitDownloadSpeed: false,
		LimitDownloadSpeed:   0,
		IsLimitUploadSpeed:   false,
		LimitUploadSpeed:     0,
		ChainId:              "tropos-1",
		Token:                "ustos",
		StratosChainUrl:      "http://127.0.0.1:1317",
		RestPort:             "",
		InternalPort:         "",
		TrafficLogInterval:   10,
		SPList:               []SPBaseInfo{{NetworkAddress: "127.0.0.1:8888"}},
	}
}

func GenDefaultConfig(filePath string) error {
	cfg := defaultConfig()

	return utils.WriteConfig(cfg, filePath)
}

func formalizePath(config2 *config) (err error) {
	//if the configuration are using default path, try to load the root path specified from flag
	if Config.AccountDir == "./accounts" {
		Config.AccountDir = filepath.Join(rootPath, Config.AccountDir)
	}
	if Config.PPListDir == "./peers" {
		Config.PPListDir = filepath.Join(rootPath, Config.PPListDir)
	}
	if Config.StorehousePath == "./storage" {
		Config.StorehousePath = filepath.Join(rootPath, Config.StorehousePath)
	}
	if Config.DownloadPath == "./download" {
		Config.DownloadPath = filepath.Join(rootPath, Config.DownloadPath)
	}

	// make the path absolute if the configured path is not the default value, won't consider the home flag
	if !filepath.IsAbs(Config.AccountDir) {
		Config.AccountDir, err = filepath.Abs(Config.AccountDir)
		if err != nil {
			return err
		}
	}
	if !filepath.IsAbs(Config.StorehousePath) {
		Config.StorehousePath, err = filepath.Abs(Config.StorehousePath)
		if err != nil {
			return err
		}
	}
	if !filepath.IsAbs(Config.DownloadPath) {
		Config.DownloadPath, err = filepath.Abs(Config.DownloadPath)
		if err != nil {
			return err
		}
	}
	if !filepath.IsAbs(Config.PPListDir) {
		Config.PPListDir, err = filepath.Abs(Config.PPListDir)
		if err != nil {
			return err
		}
	}
	return nil
}
