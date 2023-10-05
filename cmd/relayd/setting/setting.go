package setting

import (
	"os"

	"github.com/stratosnet/sds/utils"
)

var Config *config
var HomePath string

const (
	VERSION     = "v0.11.0"
	APP_VER     = 11
	MIN_APP_VER = 11
)

type AppVersion struct {
	AppVer    uint16 `toml:"app_ver"`
	MinAppVer uint16 `toml:"min_app_ver"`
	Show      string `toml:"show"`
}

type connectionRetries struct {
	Max             int `toml:"max"`
	SleepDuration   int `toml:"sleep_duration"`   // Milliseconds
	RefreshInterval int `toml:"refresh_interval"` // Seconds
}

type grpcConfig struct {
	GrpcServer string `toml:"grpc_server" comment:"Network address of the chain Eg: \"127.0.0.1:9090\""`
	Insecure   bool   `toml:"insecure"`
}

type sds struct {
	ApiPort           string            `toml:"api_port"`
	NetworkAddress    string            `toml:"network_address"`
	WebsocketPort     string            `toml:"websocket_port"`
	ConnectionRetries connectionRetries `toml:"connection_retries"`
}

type broadcast struct {
	ChannelSize int `toml:"channel_size"`
	MaxMsgPerTx int `toml:"max_msg_per_tx"`
}

type stratoschain struct {
	GrpcServer        grpcConfig        `toml:"grpc_server"`
	WebsocketServer   string            `toml:"websocket_server"`
	ConnectionRetries connectionRetries `toml:"connection_retries"`
	Broadcast         broadcast         `toml:"broadcast"`
}

type transactionsConfig struct {
	GasPrice      string  `toml:"gas_price"`
	GasAdjustment float64 `toml:"gas_adjustment"`
}

type blockchainInfoConfig struct {
	ChainId      string             `toml:"chain_id"`
	Transactions transactionsConfig `toml:"transactions"`
}

type Version struct {
	AppVer    uint16 `toml:"app_ver"`
	MinAppVer uint16 `toml:"min_app_ver"`
	Show      string `toml:"show"`
}

type keysConfig struct {
	WalletPath     string `toml:"wallet_path"`
	WalletPassword string `toml:"wallet_password"`
}

type config struct {
	BlockchainInfo blockchainInfoConfig `toml:"blockchain_info"`
	Keys           keysConfig           `toml:"keys"`
	SDS            sds                  `toml:"sds"`
	StratosChain   stratoschain         `toml:"stratos_chain"`
	Version        Version              `toml:"version"`
	Node           NodeConfig           `toml:"node" comment:"Configuration of this node"`
}
type ConnectivityConfig struct {
	RpcPort       string `toml:"rpc_port" comment:"Port for the JSON-RPC api. See https://docs.thestratos.org/docs-resource-node/sds-rpc-for-file-operation/"`
	AllowOwnerRpc bool   `toml:"allow_owner_rpc" comment:"Enable the node owner RPC API. This API can manipulate the node status and sign txs with the local wallet. Do not open this to the internet  Eg: false"`
}

type NodeConfig struct {
	Connectivity ConnectivityConfig `toml:"connectivity"`
}

func LoadConfig(path string) error {
	Config = new(config)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		utils.Log("The config at location", path, "does not exist")
		return err
	}

	err := utils.LoadTomlConfig(Config, path)
	if err != nil {
		return err
	}

	return nil
}

func defaultConfig() *config {
	return &config{
		BlockchainInfo: blockchainInfoConfig{
			ChainId: "testchain",
			Transactions: transactionsConfig{
				GasPrice:      "1000000000wei",
				GasAdjustment: 2.0,
			},
		},
		Keys: keysConfig{
			WalletPath:     "config/st1a8ngk4tjvuxneyuvyuy9nvgehkpfa38hm8mp3x.json",
			WalletPassword: "aaa",
		},
		SDS: sds{
			ApiPort:        "8081",
			NetworkAddress: "127.0.0.1",
			WebsocketPort:  "8889",
			ConnectionRetries: connectionRetries{
				Max:             100,
				SleepDuration:   3000,
				RefreshInterval: 24 * 60 * 60,
			},
		},
		StratosChain: stratoschain{
			GrpcServer: grpcConfig{
				GrpcServer: "127.0.0.1:9090",
				Insecure:   true,
			},
			WebsocketServer: "127.0.0.1:26657",
			ConnectionRetries: connectionRetries{
				Max:             100,
				SleepDuration:   3000,
				RefreshInterval: 24 * 60 * 60,
			},
			Broadcast: broadcast{
				ChannelSize: 2000,
				MaxMsgPerTx: 250,
			},
		},
		Version: Version{AppVer: APP_VER, MinAppVer: MIN_APP_VER, Show: VERSION},
		Node: NodeConfig{Connectivity: ConnectivityConfig{
			RpcPort:       "9095",
			AllowOwnerRpc: true,
		}},
	}
}

func GenDefaultConfig(filePath string) error {
	cfg := defaultConfig()

	return utils.WriteTomlConfig(cfg, filePath)
}
