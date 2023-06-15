package setting

import (
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

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

	DEFAULT_MIN_UNSUSPEND_DEPOSIT = "1stos" // 1 stos

	SPAM_THRESHOLD_SP_SIGN_LATENCY  = 60 // in second
	SPAM_THRESHOLD_SLICE_OPERATIONS = 6 * time.Hour

	SOFT_RAM_LIMIT_TIER_0     = int64(3 * units.GiB)
	SOFT_RAM_LIMIT_TIER_1     = int64(7 * units.GiB)
	SOFT_RAM_LIMIT_TIER_2     = int64(15 * units.GiB)
	SOFT_RAM_LIMIT_UNLIMITED  = math.MaxInt64
	SOFT_RAM_LIMIT_TIER_0_DEV = int64(300 * units.MiB)
	SOFT_RAM_LIMIT_TIER_1_DEV = int64(500 * units.MiB)
	SOFT_RAM_LIMIT_TIER_2_DEV = int64(700 * units.MiB)

	DEFAULT_HLS_SEGMENT_BUFFER = 4
	DEFAULT_HLS_SEGMENT_LENGTH = 10
	DEFAULT_SLICE_BLOCK_SIZE   = 33554432
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

type WebServerConfig struct {
	Path string `toml:"path"`
	Port string `toml:"port"`
}

type config struct {
	Version              AppVersion      `toml:"version"`
	DownloadPathMinLen   int             `toml:"download_path_min_len"`
	Port                 string          `toml:"port"`
	NetworkAddress       string          `toml:"network_address"`
	Debug                bool            `toml:"debug"`
	PPListDir            string          `toml:"pp_list_dir"`
	AccountDir           string          `toml:"account_dir"`
	StorehousePath       string          `toml:"storehouse_path"`
	SelfClaimedDiskSize  uint64          `toml:"self_claimed_disk_size"`
	DownloadPath         string          `toml:"download_path"`
	P2PAddress           string          `toml:"p2p_address"`
	P2PPassword          string          `toml:"p2p_password"`
	WalletAddress        string          `toml:"wallet_address"`
	WalletPassword       string          `toml:"wallet_password"`
	AutoRun              bool            `toml:"auto_run"`  // is auto login
	Internal             bool            `toml:"internal"`  // is internal net
	IsWallet             bool            `toml:"is_wallet"` // is wallet
	IsLimitDownloadSpeed bool            `toml:"is_limit_download_speed"`
	LimitDownloadSpeed   uint64          `toml:"limit_download_speed"`
	IsLimitUploadSpeed   bool            `toml:"is_limit_upload_speed"`
	LimitUploadSpeed     uint64          `toml:"limit_upload_speed"`
	ChainId              string          `toml:"chain_id"`
	StratosChainUrl      string          `toml:"stratos_chain_url"`
	Insecure             bool            `toml:"insecure"`
	GasAdjustment        float64         `toml:"gas_adjustment"`
	RestPort             string          `toml:"rest_port"`
	InternalPort         string          `toml:"internal_port"`
	RpcPort              string          `toml:"rpc_port"`
	AllowOwnerRpc        bool            `toml:"allow_owner_rpc"`
	MetricsPort          string          `toml:"metrics_port"`
	Monitor              MonitorConn     `toml:"monitor"`
	WebServer            WebServerConfig `toml:"web_server"`
	TrafficLogInterval   uint64          `toml:"traffic_log_interval"`
	MaxConnection        int             `toml:"max_connection"`
	SPList               []SPBaseInfo    `toml:"sp_list"`
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

	err = formalizePaths()
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
		Monitor: MonitorConn{
			TLS:  false,
			Cert: "",
			Key:  "",
			Port: "9501",
		},
		WebServer: WebServerConfig{
			Path: "./web",
			Port: "8081",
		},
		TrafficLogInterval: 10,
		SPList:             []SPBaseInfo{{NetworkAddress: "127.0.0.1:8888"}},
		AllowOwnerRpc:      false,
	}
}

func GenDefaultConfig() error {
	Config = defaultConfig()
	return FlushConfig()
}

// formalizePaths checks if the configuration is using default paths. If so, add in the node root path and make it absolute
func formalizePaths() (err error) {
	defaultValues := defaultConfig()

	Config.AccountDir, err = formalizePath(Config.AccountDir, defaultValues.AccountDir)
	if err != nil {
		return err
	}

	Config.PPListDir, err = formalizePath(Config.PPListDir, defaultValues.PPListDir)
	if err != nil {
		return err
	}

	Config.StorehousePath, err = formalizePath(Config.StorehousePath, defaultValues.StorehousePath)
	if err != nil {
		return err
	}

	Config.DownloadPath, err = formalizePath(Config.DownloadPath, defaultValues.DownloadPath)
	if err != nil {
		return err
	}

	Config.WebServer.Path, err = formalizePath(Config.WebServer.Path, defaultValues.WebServer.Path)
	if err != nil {
		return err
	}

	return nil
}

func formalizePath(path, defaultValue string) (newPath string, err error) {
	newPath = path
	if path == defaultValue {
		newPath = filepath.Join(rootPath, path)
	}

	// make the path absolute if the configured path is not the default value, won't consider the home flag
	if !filepath.IsAbs(newPath) {
		newPath, err = filepath.Abs(newPath)
	}
	return newPath, err
}

func GetDiskSizeSoftCap(actualTotal uint64) uint64 {
	selfClaimedDiskSize := Config.SelfClaimedDiskSize
	if selfClaimedDiskSize != 0 && selfClaimedDiskSize < actualTotal {
		return selfClaimedDiskSize
	}
	return actualTotal
}
