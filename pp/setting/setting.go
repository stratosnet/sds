package setting

import (
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/alecthomas/units"
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/pp"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/relay/stratoschain/grpc"
	"github.com/stratosnet/sds/utils"
)

const (
	Version        = "v0.9.0"
	APP_VER        = 9
	MIN_APP_VER    = 9
	HD_PATH        = "m/44'/606'/0'/0/0"
	PP_SERVER_TYPE = "tcp4"

	// REPORTDHTIME 1 hour
	REPORTDHTIME = 60 * 60
	// NodeReportIntervalSec Interval of node stat report
	NodeReportIntervalSec   = 300 // in seconds
	NodeReportCheckInterval = 500 // in num of heights
	WeightDeductionInterval = 200 // interval for weight deduction in heights
	PpLatencyCheckInterval  = 60  // interval for checking the latency to next PP

	// MAXDATA max size of a piece in a slice
	MAXDATA        = 1024 * 1024 * 3
	MAX_SLICE_SIZE = 1024 * 1024 * 32
	HTTPTIMEOUT    = 20 // seconds
	IMAGEPATH      = "./images/"
	VIDEOPATH      = "./videos"

	STREAM_CACHE_MAXSLICE = 2

	FILE_SLICE_DOWNLOAD_BATCH_SIZE         = 20
	UPDATE_LATEST_STATUS_REPORT_BATCH_SIZE = 20

	DEFAULT_MAX_CONNECTION = 1000

	DEFAULT_MIN_UNSUSPEND_STAKE    = "1stos" // 1 stos
	SPAM_THRESHOLD_SP_SIGN_LATENCY = 120     // in second

	SOFT_RAM_LIMIT_TIER_0    = int64(3 * units.GiB)
	SOFT_RAM_LIMIT_TIER_1    = int64(7 * units.GiB)
	SOFT_RAM_LIMIT_TIER_2    = int64(15 * units.GiB)
	SOFT_RAM_LIMIT_UNLIMITED = math.MaxInt64
)

var (
	Config     *config
	ConfigPath string
	rootPath   string

	// ImageMap download image hash map
	ImageMap = &sync.Map{}

	// DownloadProgressMap download progress map
	DownloadProgressMap = &sync.Map{}

	IsLoad bool

	UploadTaskIDMap   = &sync.Map{}
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

	UpChan = make(chan string, 100)

	TrafficLogPath string
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

type MonitorConn struct {
	TLS  bool   `toml:"tls"`
	Cert string `toml:"cert"`
	Key  string `toml:"key"`
	Port string `toml:"port"`
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
	SelfClaimedDiskSize  uint64       `toml:"self_claimed_disk_size"`
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
	StratosChainUrl      string       `toml:"stratos_chain_url"`
	Insecure             bool         `toml:"insecure"`
	GasAdjustment        float64      `toml:"gas_adjustment"`
	RestPort             string       `toml:"rest_port"`
	InternalPort         string       `toml:"internal_port"`
	RpcPort              string       `toml:"rpc_port"`
	AllowOwnerRpc        bool         `toml:"allow_owner_rpc"`
	MetricsPort          string       `toml:"metrics_port"`
	Monitor              MonitorConn  `toml:"monitor"`
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

	err = formalizePath()
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
	grpc.URL = Config.StratosChainUrl
	grpc.INSECURE = Config.Insecure
	pp.ALLOW_OWNER_RPC = Config.AllowOwnerRpc

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

func CheckLogin() bool {
	if WalletAddress == "" {
		utils.ErrorLog("please login")
		return false
	}
	return true
}

func SetConfig(key string, value interface{}) error {
	tomlTree, err := toml.LoadFile(ConfigPath)
	if err != nil {
		return err
	}

	// Read existing value
	if !tomlTree.Has(key) {
		return errors.Errorf("Key [%v] doesn't exist", key)
	}
	existingValue := tomlTree.Get(key)
	switch existingValue.(type) {
	case *toml.Tree:
		return errors.Errorf("Key [%v] is a subtree. It cannot be edited directly", key)
	case []*toml.Tree:
		return errors.Errorf("Key [%v] is a subtree. It cannot be edited directly", key)
	default:
		if existingValue == value {
			return nil
		}
		tomlTree.Set(key, value)
	}

	// Check if change is valid
	data, err := tomlTree.Marshal()
	if err != nil {
		return err
	}
	if err = toml.Unmarshal(data, &config{}); err != nil {
		return err
	}

	// Save changes to file
	err = os.WriteFile(ConfigPath, data, 0644)
	if err != nil {
		return err
	}

	// Reload config object
	return LoadConfig(ConfigPath)
}

func FlushConfig() error {
	return utils.WriteTomlConfig(Config, ConfigPath)
}

func defaultConfig() *config {
	return &config{
		Version:              AppVersion{AppVer: APP_VER, MinAppVer: MIN_APP_VER, Show: Version},
		DownloadPathMinLen:   88,
		Port:                 "18081",
		NetworkAddress:       "127.0.0.1",
		Debug:                false,
		PPListDir:            "./peers",
		AccountDir:           "./accounts",
		StorehousePath:       "./storage",
		SelfClaimedDiskSize:  1099511627776, // 1TB by default
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
		ChainId:              "tropos-5",
		StratosChainUrl:      "127.0.0.1:9090",
		Insecure:             true,
		GasAdjustment:        1.3,
		RestPort:             "",
		InternalPort:         "",
		TrafficLogInterval:   10,
		SPList:               []SPBaseInfo{{NetworkAddress: "127.0.0.1:8888"}},
		AllowOwnerRpc:        false,
	}
}

func GenDefaultConfig() error {
	Config = defaultConfig()
	return FlushConfig()
}

func formalizePath() (err error) {
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

func GetDiskSizeSoftCap(actualTotal uint64) uint64 {
	selfClaimedDiskSize := Config.SelfClaimedDiskSize
	if selfClaimedDiskSize != 0 && selfClaimedDiskSize < actualTotal {
		return selfClaimedDiskSize
	}
	return actualTotal
}
