package setting

import (
	"github.com/stratosnet/sds/utils"
)

var Config *config
var HomePath string

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
	RestServer        string            `toml:"rest_server"`
	WebsocketServer   string            `toml:"websocket_server"`
	ConnectionRetries connectionRetries `toml:"connection_retries"`
	Broadcast         broadcast         `toml:"broadcast"`
}

type transactionsConfig struct {
	Fee           string  `toml:"fee"`
	GasAdjustment float64 `toml:"gas_adjustment"`
}

type blockchainInfoConfig struct {
	ChainId      string             `toml:"chain_id"`
	Transactions transactionsConfig `toml:"transactions"`
}

type Version struct {
	AppVer    uint16 `toml:"app_ver"`
	MinAppVer uint16 `toml:"min_app_ver"`
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
	err := utils.LoadTomlConfig(Config, path)
	if err != nil {
		return err
	}

	return nil
}
