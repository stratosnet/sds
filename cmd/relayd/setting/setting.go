package setting

import (
	"os"

	"github.com/stratosnet/sds/utils"
)

var Config *config
var HomePath string

const (
	VERSION     = "v0.9.0"
	APP_VER     = 9
	MIN_APP_VER = 9
)

type AppVersion struct {
	AppVer    uint16 `toml:"app_ver"`
	MinAppVer uint16 `toml:"min_app_ver"`
	Show      string `toml:"show"`
}

type connectionRetries struct {
	Max           int `toml:"max"`
	SleepDuration int `toml:"sleep_duration"`
}

type sds struct {
	ApiPort           string            `toml:"api_port"`
	ClientPort        string            `toml:"client_port"`
	NetworkAddress    string            `toml:"network_address"`
	WebsocketPort     string            `toml:"websocket_port"`
	ConnectionRetries connectionRetries `toml:"connection_retries"`
}

type broadcast struct {
	ChannelSize int `toml:"channel_size"`
	MaxMsgPerTx int `toml:"max_msg_per_tx"`
}

type stratoschain struct {
	GrpcServer        string            `toml:"grpc_server"`
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
			ChainId: "testchain_1-1",
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
			ClientPort:     "8088",
			NetworkAddress: "127.0.0.1",
			WebsocketPort:  "8889",
			ConnectionRetries: connectionRetries{
				Max:           100,
				SleepDuration: 3000,
			},
		},
		StratosChain: stratoschain{
			GrpcServer:      "http://127.0.0.1:9090",
			WebsocketServer: "127.0.0.1:26657",
			ConnectionRetries: connectionRetries{
				Max:           100,
				SleepDuration: 3000,
			},
			Broadcast: broadcast{
				ChannelSize: 2000,
				MaxMsgPerTx: 250,
			},
		},
		Version: Version{AppVer: APP_VER, MinAppVer: MIN_APP_VER, Show: VERSION},
	}
}

func GenDefaultConfig(filePath string) error {
	cfg := defaultConfig()

	return utils.WriteTomlConfig(cfg, filePath)
}
