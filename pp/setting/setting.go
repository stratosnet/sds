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
	Version     = "v0.8.1"
	APP_VER     = 8
	MIN_APP_VER = 8
	HD_PATH     = "m/44'/606'/0'/0/0"

	// REPORTDHTIME 1 hour
	REPORTDHTIME = 60 * 60
	// NodeReportIntervalSec Interval of node stat report
	NodeReportIntervalSec   = 300 // in seconds
	NodeReportCheckInterval = 500 // in num of heights
	WeightDeductionInterval = 200 // interval for weight deduction in heights

	// MAXDATA max slice size
	MAXDATA = 1024 * 1024 * 3
	// HTTPTIMEOUT  HTTPTIMEOUT second
	HTTPTIMEOUT = 20
	// IMAGEPATH
	IMAGEPATH = "./images/"
	VIDEOPATH = "./videos"

	STREAM_CACHE_MAXSLICE = 2

	FILE_SLICE_DOWNLOAD_BATCH_SIZE         = 20
	UPDATE_LATEST_STATUS_REPORT_BATCH_SIZE = 20

	DEFAULT_MAX_CONNECTION = 1000
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

type AppVersion struct {
	AppVer    uint16 `toml:"app_ver"`
	MinAppVer uint16 `toml:"min_app_ver"`
	Show      string `toml:"show"`
}

type SPBaseInfo struct {
	P2PAddress     string `toml:"p2p_address"`
	P2PPublicKey   string `toml:"p2p_public_key"`
	NetworkAddress string `toml:"network_address"`
}

type config struct {
	Version              AppVersion   `toml:"version"`
	DownloadPathMinLen   int          `toml:"download_path_min_len"`
	Port                 string       `toml:"port"`
	NetworkAddress       string       `toml:"network_address"`
	Debug                bool         `toml:"debug"`
	PPListDir            string       `toml:"pp_list_dir"`
	AccountDir           string       `toml:"account_dir"`
	StorehousePath       string       `toml:"storehouse_path"`
	DownloadPath         string       `toml:"download_path"`
	P2PAddress           string       `toml:"p2p_address"`
	P2PPassword          string       `toml:"p2p_password"`
	WalletAddress        string       `toml:"wallet_address"`
	WalletPassword       string       `toml:"wallet_password"`
	AutoRun              bool         `toml:"auto_run"`  // is auto login
	Internal             bool         `toml:"internal"`  // is internal net
	IsWallet             bool         `toml:"is_wallet"` // is wallet
	IsLimitDownloadSpeed bool         `toml:"is_limit_download_speed"`
	LimitDownloadSpeed   uint64       `toml:"limit_download_speed"`
	IsLimitUploadSpeed   bool         `toml:"is_limit_upload_speed"`
	LimitUploadSpeed     uint64       `toml:"limit_upload_speed"`
	ChainId              string       `toml:"chain_id"`
	Token                string       `toml:"token"`
	StratosChainUrl      string       `toml:"stratos_chain_url"`
	RestPort             string       `toml:"rest_port"`
	InternalPort         string       `toml:"internal_port"`
	TrafficLogInterval   uint64       `toml:"traffic_log_interval"`
	MaxConnection        int          `toml:"max_connection"`
	SPList               []SPBaseInfo `toml:"sp_list"`
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
	err := utils.LoadTomlConfig(Config, configPath)
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
	found, isString := utils.CheckStructField(key, Config)
	if !found {
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
	for _, str := range contentStrs {
		ss := strings.Split(str, " ")
		if len(ss) > 0 && ss[0] == key {
			if key == "download_path" {
				if ostype == "windows" {
					value = value + `\`
				} else {
					value = value + `/`
				}
			}
			ns := ""
			if isString {
				ns = key + " = '" + value + "'"
			} else {
				ns = key + " = " + value
			}
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
		Version:              AppVersion{AppVer: APP_VER, MinAppVer: MIN_APP_VER, Show: Version},
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
		ChainId:              "tropos-4",
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

	return utils.WriteTomlConfig(cfg, filePath)
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
