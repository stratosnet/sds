package setting

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/utils"
)

// REPROTDHTIME 1 hour
const REPROTDHTIME = 60 * 60

// MAXDATA max slice size
const MAXDATA = 1024 * 1024 * 3

// HTTPTIMEOUT  HTTPTIMEOUT second
const HTTPTIMEOUT = 10

// FILEHASHLEN
const FILEHASHLEN = 64

// IMAGEPATH 保存图片路径
var IMAGEPATH = "./images/"

// ImageMap download image hash map
var ImageMap = &sync.Map{}

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

// DownLoadTaskIDMap
var DownLoadTaskIDMap = &sync.Map{}

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
	Password                    string `yaml:"Password"`
	Account                     string `yaml:"Account"`
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
	Token                       string `yaml:"Token"`
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
	data, err := utils.ECCSign([]byte(str), PrivateKey)
	utils.DebugLog("GetSign == ", data)
	if err != nil {
		utils.ErrorLog("GetSign", err)
		return nil
	}
	return data
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
