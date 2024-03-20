package setting

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/tx-client/grpc"
)

var (
	Config         *config
	ConfigPath     string
	rootPath       string
	TrafficLogPath string

	// ImageMap download image hash map
	ImageMap = &sync.Map{}

	// DownloadProgressMap download progress map
	DownloadProgressMap = &sync.Map{}

	ostype    = runtime.GOOS
	IsWindows bool

	meta_p2p    string
	meta_pubkey string
	meta_net    string
)

func init() {
	meta_entry := []struct {
		P2p    string
		PubKey string
		Net    string
	}{
		{"stsds1z96pm5ls0ff2y7y8adpy6r3l8jqeaud7envnqv",
			"stsdspub1lf769k20k36e4gvnewcwdtfudzj95qk45d5f0p300jmr7e6y73zsdyh25y",
			"34.82.40.37:8888",
		},
		{"stsds10kmygjv7e2t39f6jka6445q20e9lv4a7u3qex3",
			"stsdspub1srn3qetarx3x6f2x9wqfv3nh2aufxv03ncl5v6jkmyg666scvz6s4xgprq",
			"34.85.35.57:8888",
		},
		{"stsds1ypxg8sj5vn4s4v0w965g4r9g3pt3vlz6wyzx0f",
			"stsdspub1y6exsr8snwz65ev3pzq6k3yfy2ku3kexqdd0en35dnr8mxc9w6sq5jg6lf",
			"34.34.149.18:8888",
		},
	}
	rand := time.Now().UnixMilli() % 3
	meta_p2p = meta_entry[rand].P2p
	meta_pubkey = meta_entry[rand].PubKey
	meta_net = meta_entry[rand].Net

}

type VersionConfig struct {
	AppVer    uint16 `toml:"app_ver" comment:"App version number. Eg: 11"`
	MinAppVer uint16 `toml:"min_app_ver" comment:"Network connections from nodes below this version number will be rejected. Eg: 11"`
	Show      string `toml:"show" comment:"Formatted version number. Eg: \"v0.11.0\""`
}

type BlockchainConfig struct {
	ChainId       string  `toml:"chain_id" comment:"ID of the chain Eg: \"stratos-1\""`
	GasAdjustment float64 `toml:"gas_adjustment" comment:"Multiplier for the simulated tx gas cost Eg: 1.5"`
	Insecure      bool    `toml:"insecure" comment:"Connect to the chain using an insecure connection (no TLS) Eg: true"`
	GrpcServer    string  `toml:"grpc_server" comment:"Network address of the chain grpc Eg: \"127.0.0.1:9090\""`
}

type HomeConfig struct {
	AccountsPath string `toml:"accounts_path" comment:"Key files (wallet and P2P key). Eg: \"./accounts\""`
	DownloadPath string `toml:"download_path" comment:"Where downloaded files will go. Eg: \"./download\""`
	PeersPath    string `toml:"peers_path" comment:"The list of peers (other sds nodes). Eg: \"./peers\""`
	StoragePath  string `toml:"storage_path" comment:"Where files are stored. Eg: \"./storage\""`
}

type KeysConfig struct {
	P2PAddress         string `toml:"p2p_address" comment:"Address of the P2P key. Eg: \"stsdsxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\""`
	P2PPassword        string `toml:"p2p_password"`
	WalletAddress      string `toml:"wallet_address" comment:"Address of the stratos wallet. Eg: \"stxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\""`
	WalletPassword     string `toml:"wallet_password"`
	BeneficiaryAddress string `toml:"beneficiary_address" comment:"Address for receiving reward. Eg: \"stxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\""`
}

type ConnectivityConfig struct {
	SeedMetaNode   SPBaseInfo `toml:"seed_meta_node" comment:"The first meta node to connect to when starting the node"`
	Internal       bool       `toml:"internal" comment:"Is the node running on an internal network? Eg: false"`
	NetworkAddress string     `toml:"network_address" comment:"IP address of the node. Eg: \"127.0.0.1\""`
	NetworkPort    string     `toml:"network_port" comment:"Main port for communication on the network. Must be open to the internet. Eg: \"18081\""`
	MetricsPort    string     `toml:"metrics_port" comment:"Port for prometheus metrics"`
	RpcPort        string     `toml:"rpc_port" comment:"Port for the JSON-RPC api. See https://docs.thestratos.org/docs-resource-node/sds-rpc-for-file-operation/"`
	RpcNamespaces  string     `toml:"rpc_namespaces" comment:"Namespaces enabled in the RPC API. Eg: \"user,owner\"""`
}

type NodeConfig struct {
	Debug        bool               `toml:"debug" comment:"Should debug info be printed out in logs? Eg: false"`
	MaxDiskUsage uint64             `toml:"max_disk_usage" comment:"When not 0, limit disk usage to this amount (in megabytes) Eg: 7629394 = 8 * 1000 * 1000 * 1000 * 1000 / 1024 / 1024  (8TB) "`
	Connectivity ConnectivityConfig `toml:"connectivity"`
}

type MonitorConfig struct {
	TLS            bool     `toml:"tls" comment:"Should the monitor server use TLS? Eg: false"`
	CertFilePath   string   `toml:"cert_file_path" comment:"Path to the TLS certificate file"`
	KeyFilePath    string   `toml:"key_file_path" comment:"Path to the TLS private key file"`
	Port           string   `toml:"port" comment:"Port used for the monitor websocket connection. It's the monitor UI that uses this port, not the person accessing the UI in a browser"`
	AllowedOrigins []string `toml:"allowed_origins" comment:"List of IPs that are allowed to connect to the monitor websocket port. This is used to decide which IP can connect their monitor to the node, NOT to decide who can view the monitor UI page."`
}

type TrafficConfig struct {
	LogInterval     uint64 `toml:"log_interval" comment:"Interval at which traffic is logged (in seconds) Eg: 10"`
	MaxConnections  int    `toml:"max_connections" comment:"Max number of concurrent network connections. Eg: 1000"`
	MaxDownloadRate uint64 `toml:"max_download_rate" comment:"Max number of download messages received per second (per connection). 0 Means unlimited. 1000 ≈ 1MB/sec. Eg: 1000"`
	MaxUploadRate   uint64 `toml:"max_upload_rate" comment:"Max number of upload messages sent per second (per connection). 0 Means unlimited. 1000 ≈ 1MB/sec. Eg: 1000"`
}

type StreamingConfig struct {
	InternalPort string `toml:"internal_port" comment:"Port for the internal HTTP server"`
	RestPort     string `toml:"rest_port" comment:"Port for the REST server"`
}

type WebServerConfig struct {
	Path           string `toml:"path" comment:"Location of the web server files Eg: \"./web\""`
	Port           string `toml:"port" comment:"Port where the web server is hosted with sdsweb. If the port is opened and token_on_startup is true, anybody who loads the monitor UI will have full access to the monitor"`
	TokenOnStartup bool   `toml:"token_on_startup" comment:"Automatically enter monitor token when opening the monitor UI. This should be false if the web_server port is opened to internet and you don't want public access to your node monitor'"`
}

type config struct {
	Version    VersionConfig    `toml:"version"`
	Blockchain BlockchainConfig `toml:"blockchain" comment:"Configuration of the connection to the Stratos blockchain"`
	Home       HomeConfig       `toml:"home" comment:"Structure of the home folder. Default paths (eg: \"./storage\" become relative to the node home. Other paths are relative to the working directory"`
	Keys       KeysConfig       `toml:"keys"`
	Node       NodeConfig       `toml:"node" comment:"Configuration of this node"`
	Monitor    MonitorConfig    `toml:"monitor" comment:"Configuration for the monitor server"`
	Streaming  StreamingConfig  `toml:"streaming" comment:"Configuration for video streaming"`
	Traffic    TrafficConfig    `toml:"traffic"`
	WebServer  WebServerConfig  `toml:"web_server" comment:"Configuration for the web server (when running sdsweb)"`
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

	cf.SetMaxDownloadRate(Config.Traffic.MaxDownloadRate)
	cf.SetMaxUploadRate(Config.Traffic.MaxUploadRate)

	// todo: we shouldn't call grpc package to setup a global variable
	grpc.SERVER = Config.Blockchain.GrpcServer
	grpc.INSECURE = Config.Blockchain.Insecure

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

	updatedConfig := &config{}
	if err = toml.Unmarshal(data, updatedConfig); err != nil {
		return err
	}

	// Save changes to file
	Config = updatedConfig
	if err = FlushConfig(); err != nil {
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
		Version: VersionConfig{AppVer: AppVersion, MinAppVer: MinAppVersion, Show: Version},
		Blockchain: BlockchainConfig{
			ChainId:       "stratos-1",
			GasAdjustment: 1.5,
			Insecure:      false,
			GrpcServer:    "grpc.thestratos.org:443",
		},
		Home: HomeConfig{
			AccountsPath: "./accounts",
			DownloadPath: "./download",
			PeersPath:    "./peers",
			StoragePath:  "./storage",
		},
		Keys: KeysConfig{
			P2PAddress:         "",
			P2PPassword:        "",
			WalletAddress:      "",
			WalletPassword:     "",
			BeneficiaryAddress: "",
		},
		Node: NodeConfig{
			Debug:        false,
			MaxDiskUsage: 8 * 1000 * 1000 * 1000 * 1000 / 1024 / 1024, // 8TB,
			Connectivity: ConnectivityConfig{
				SeedMetaNode: SPBaseInfo{
					P2PAddress:     meta_p2p,
					P2PPublicKey:   meta_pubkey,
					NetworkAddress: meta_net,
				},
				Internal:       false,
				NetworkAddress: "127.0.0.1",
				NetworkPort:    "18081",
				MetricsPort:    "18181",
				RpcPort:        "18281",
				RpcNamespaces:  "user",
			},
		},
		Monitor: MonitorConfig{
			TLS:            false,
			CertFilePath:   "",
			KeyFilePath:    "",
			Port:           "18381",
			AllowedOrigins: []string{"localhost"},
		},
		Streaming: StreamingConfig{
			InternalPort: "18481",
			RestPort:     "18581",
		},
		Traffic: TrafficConfig{
			LogInterval:     10,
			MaxConnections:  DefaultMaxConnections,
			MaxDownloadRate: 0,
			MaxUploadRate:   0,
		},
		WebServer: WebServerConfig{
			Path:           "./web",
			Port:           "18681",
			TokenOnStartup: false,
		},
	}
}

func GenDefaultConfig() error {
	Config = defaultConfig()
	return FlushConfig()
}

// formalizePaths checks if the configuration is using default paths, and if so, add in the node root path. It also makes all paths absolute
func formalizePaths() (err error) {
	defaultValues := defaultConfig()

	Config.Home.AccountsPath, err = formalizePath(Config.Home.AccountsPath, defaultValues.Home.AccountsPath)
	if err != nil {
		return err
	}

	Config.Home.PeersPath, err = formalizePath(Config.Home.PeersPath, defaultValues.Home.PeersPath)
	if err != nil {
		return err
	}

	Config.Home.StoragePath, err = formalizePath(Config.Home.StoragePath, defaultValues.Home.StoragePath)
	if err != nil {
		return err
	}

	Config.Home.DownloadPath, err = formalizePath(Config.Home.DownloadPath, defaultValues.Home.DownloadPath)
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

	// Make the path absolute (it won't consider the home flag)
	if !filepath.IsAbs(newPath) {
		newPath, err = filepath.Abs(newPath)
	}
	return newPath, err
}

func GetDiskSizeSoftCap(actualTotal uint64) uint64 {
	maxDiskBytes := Config.Node.MaxDiskUsage * 1024 * 1024 // MB to B
	if maxDiskBytes != 0 && maxDiskBytes < actualTotal {
		return maxDiskBytes
	}
	return actualTotal
}

func GetDataBufferSize() int {
	i, err := strconv.ParseInt(os.Getenv("PPD_DATA_BUF_SIZE"), 10, 0)
	if err != nil {
		return DEFAULT_DATA_BUFFER_POOL_SIZE
	}
	return int(i)
}
