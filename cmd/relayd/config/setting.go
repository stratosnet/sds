package setting

import (
	"github.com/stratosnet/sds/relay/stratoschain/prefix"
	"github.com/stratosnet/sds/utils"
)

type connectionRetries struct {
	Max           int `yaml:"max"`
	SleepDuration int `yaml:"sleepDuration"`
}

type sds struct {
	ClientPort        string            `yaml:"clientPort"`
	NetworkAddress    string            `yaml:"networkAddress"`
	WebsocketPort     string            `yaml:"websocketPort"`
	ConnectionRetries connectionRetries `yaml:"connectionRetries"`
}

type stratoschain struct {
	NetworkAddress string `yaml:"networkAddress"`
	RestPort       string `yaml:"restPort"`
	WebsocketPort  string `yaml:"websocketPort"`
}

type config struct {
	BlockchainInfo blockchainInfoConfig `yaml:"blockchainInfo"`
	SDS            sds                  `yaml:"sds"`
	StratosChain   stratoschain         `yaml:"stratosChain"`
}

type blockchainInfoConfig struct {
	AddressPrefix    string `yaml:"addressPrefix"`
	P2PAddressPrefix string `yaml:"p2pAddressPrefix"`
	ChainId          string `yaml:"chainId"`
	Token            string `yaml:"token"`
}

var Config *config

func LoadConfig(path string) {
	Config = new(config)
	utils.LoadYamlConfig(Config, path)
	prefix.SetConfig(Config.BlockchainInfo.AddressPrefix)
}
